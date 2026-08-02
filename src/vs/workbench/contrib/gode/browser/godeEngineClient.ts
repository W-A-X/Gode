/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { IPos, IRange, IGodeCommand, IGodeEvent, ITokenLine } from '../common/godeProtocol.js';

/**
 * WebSocket client for the gode-engine offscreen rendering process.
 *
 * Commands are JSON messages; frames arrive as events carrying RGBA pixels
 * (base64-encoded over the wire), which are drawn onto a canvas.
 */
export class GodeEngineClient {

	private readonly _canvas: HTMLCanvasElement;
	private readonly _ctx: CanvasRenderingContext2D;
	private _ws: WebSocket | null = null;
	private _reconnectTimer: number | null = null;
	private _queuedCommands: IGodeCommand[] = [];
	private _onDidEditCbs: ((range: IRange | null, text: string) => void)[] = [];
	private _onDidChangeSelectionCbs: ((anchor: IPos, active: IPos) => void)[] = [];
	private _pendingViewport: { width: number; height: number; scale: number } | null = null;
	private _lastSentViewport: { width: number; height: number; scale: number } | null = null;
	private _disposed = false;

	constructor(
		private readonly _port: number,
		canvas: HTMLCanvasElement,
		ctx: CanvasRenderingContext2D,
	) {
		this._canvas = canvas;
		this._ctx = ctx;
		this._connect();
	}

	public dispose(): void {
		this._disposed = true;
		if (this._reconnectTimer !== null) {
			window.clearTimeout(this._reconnectTimer);
			this._reconnectTimer = null;
		}
		this._ws?.close();
		this._ws = null;
	}

	public onDidEdit(cb: (range: IRange | null, text: string) => void): void {
		this._onDidEditCbs.push(cb);
	}

	public onDidChangeSelection(cb: (anchor: IPos, active: IPos) => void): void {
		this._onDidChangeSelectionCbs.push(cb);
	}

	public openDocument(text: string): void {
		this._send({ cmd: 'open_document', text });
	}

	/** Replaces the engine's document text wholesale (VS Code model -> engine sync). */
	public setText(text: string): void {
		this.openDocument(text);
	}

	public focus(): void {
		this._send({ cmd: 'focus' });
	}

	public setTokens(lines: readonly ITokenLine[]): void {
		this._send({ cmd: 'set_tokens', tokens: lines });
	}

	/** Sets the width (device pixels) reserved for the breakpoint gutter. */
	public setGlyphMarginWidth(width: number): void {
		this._send({ cmd: 'set_glyph_margin_width', glyph_margin_width: width });
	}

	/** Sets the 1-based line numbers that carry a breakpoint marker. */
	public setBreakpoints(lines: readonly number[]): void {
		this._send({ cmd: 'set_breakpoints', breakpoints: lines });
	}

	public setViewport(width: number, height: number, scale: number = 1): void {
		this._pendingViewport = { width, height, scale };
		if (this._ws?.readyState === WebSocket.OPEN) {
			// VS Code calls render() (and thus setViewport) on every frame; skip
			// when unchanged so we don't spam the engine and trigger needless
			// Resize work.
			if (this._lastSentViewport
				&& this._lastSentViewport.width === width
				&& this._lastSentViewport.height === height
				&& this._lastSentViewport.scale === scale) {
				return;
			}
			this._send({ cmd: 'set_viewport', width, height, scale });
			this._lastSentViewport = { width, height, scale };
		}
	}

	public setSelection(anchor: IPos, active: IPos): void {
		this._send({ cmd: 'set_selection', anchor, active });
	}

	public requestFrame(): void {
		// Frames are pushed by the engine after every command; nothing to do.
	}

	public sendKey(keyType: 'press' | 'release' | 'repeat', e: KeyboardEvent): void {
		const key = this._translateKey(e.key);
		if (!key) {
			return;
		}
		this._send({
			cmd: 'input',
			type: 'key',
			key: {
				key_type: keyType,
				key: key.name,
				rune: keyType === 'press' ? (key.rune ?? '') : '',
				shift: e.shiftKey,
				ctrl: e.metaKey || e.ctrlKey,
				alt: e.altKey,
				super: e.metaKey,
			}
		});
	}

	public sendMouse(mouseType: 'press' | 'release' | 'move' | 'drag' | 'double_click', e: MouseEvent): void {
		const rect = this._canvas.getBoundingClientRect();
		const dpr = window.devicePixelRatio || 1;
		// Scale mouse coordinates to match the engine's DPI-scaled coordinate system.
		this._send({
			cmd: 'input',
			type: 'mouse',
			mouse: {
				mouse_type: mouseType,
				button: e.button === 2 ? 'right' : e.button === 1 ? 'middle' : 'left',
				x: (e.clientX - rect.left) * dpr,
				y: (e.clientY - rect.top) * dpr,
				shift: e.shiftKey,
				ctrl: e.metaKey || e.ctrlKey,
				alt: e.altKey,
				super: e.metaKey,
			}
		});
	}

	public sendWheel(e: WheelEvent): void {
		this._send({
			cmd: 'input',
			type: 'wheel',
			wheel: { dx: e.deltaX, dy: e.deltaY }
		});
	}

	/**
	 * Send pre-normalized pixel wheel deltas. The host normalizes deltaMode and
	 * applies VS Code's scroll sensitivity multipliers before calling this, so
	 * the engine scrolls by raw pixels (see GodeView._normalizeWheelDelta).
	 */
	public sendWheelDelta(dx: number, dy: number): void {
		this._send({
			cmd: 'input',
			type: 'wheel',
			wheel: { dx, dy }
		});
	}

	// --- internals ---

	private _connect(): void {
		if (this._disposed) {
			return;
		}
		try {
			this._ws = new WebSocket(`ws://127.0.0.1:${this._port}`);
		} catch (err) {
			this._scheduleReconnect();
			return;
		}
		this._ws.onopen = () => {
			for (const cmd of this._queuedCommands) {
				this._ws!.send(JSON.stringify(cmd));
			}
			this._queuedCommands = [];
			// Reset the viewport dedup state so the fresh connection is told the
			// current viewport (and so the next setViewport actually sends).
			this._lastSentViewport = null;
			if (this._pendingViewport) {
				this._send({ cmd: 'set_viewport', ...this._pendingViewport });
				this._lastSentViewport = { ...this._pendingViewport };
			}
		};
		this._ws.onmessage = (ev) => this._handleMessage(ev.data);
		this._ws.onclose = () => {
			this._ws = null;
			this._scheduleReconnect();
		};
		this._ws.onerror = () => {
			this._ws?.close();
		};
	}

	private _scheduleReconnect(): void {
		if (this._disposed || this._reconnectTimer !== null) {
			return;
		}
		this._reconnectTimer = window.setTimeout(() => {
			this._reconnectTimer = null;
			this._connect();
		}, 1000);
	}

	private _send(cmd: IGodeCommand): void {
		if (!this._ws || this._ws.readyState !== WebSocket.OPEN) {
			this._queuedCommands.push(cmd);
			return;
		}
		this._ws.send(JSON.stringify(cmd));
	}

	private _handleMessage(raw: any): void {
		let ev: IGodeEvent;
		try {
			ev = typeof raw === 'string' ? JSON.parse(raw) : raw;
		} catch {
			return;
		}
		switch (ev.evt) {
			case 'frame': {
				const data = this._decodeFrame(ev.data);
				if (data && ev.width && ev.height) {
					if (this._canvas.width !== ev.width || this._canvas.height !== ev.height) {
						this._canvas.width = ev.width;
						this._canvas.height = ev.height;
					}
					const imageData = new ImageData(new Uint8ClampedArray(data), ev.width, ev.height);
					this._ctx.putImageData(imageData, 0, 0);
				}
				if (ev.anchor && ev.active) {
					this._onDidChangeSelectionCbs.forEach(cb => cb(ev.anchor!, ev.active!));
				}
				break;
			}
			case 'edited': {
				this._onDidEditCbs.forEach(cb => cb(ev.range ?? null, ev.edit_text ?? ''));
				break;
			}
			case 'ready':
			case 'pong':
			case 'content':
				break;
		}
	}

	private _decodeFrame(data: string | Uint8Array | undefined): Uint8Array | null {
		if (!data) {
			return null;
		}
		if (typeof data === 'string') {
			// base64
			const bin = atob(data);
			const bytes = new Uint8Array(bin.length);
			for (let i = 0; i < bin.length; i++) {
				bytes[i] = bin.charCodeAt(i);
			}
			return bytes;
		}
		return new Uint8Array(data);
	}

	// --- key translation ---

	private _translateKey(key: string): { name: string; rune?: string } | null {
		if (key.length === 1) {
			if (key >= 'a' && key <= 'z') {
				return { name: key.toUpperCase(), rune: key };
			}
			if (key >= 'A' && key <= 'Z') {
				// key already reflects Shift/CapsLock ('A' for Shift+a). The
				// rune must be the typed character, not a forced lowercase,
				// or uppercase letters would come out lowercase.
				return { name: key, rune: key };
			}
			if (key >= '0' && key <= '9') {
				// Digits must carry a rune; without one the engine drops them.
				return { name: key, rune: key };
			}
			// punctuation and symbols
			return { name: key, rune: key };
		}
		switch (key) {
			case 'ArrowUp': return { name: 'Up' };
			case 'ArrowDown': return { name: 'Down' };
			case 'ArrowLeft': return { name: 'Left' };
			case 'ArrowRight': return { name: 'Right' };
			case 'Home': return { name: 'Home' };
			case 'End': return { name: 'End' };
			case 'PageUp': return { name: 'PageUp' };
			case 'PageDown': return { name: 'PageDown' };
			case 'Enter': return { name: 'Enter', rune: '\n' };
			case 'Backspace': return { name: 'Backspace' };
			case 'Delete': return { name: 'Delete' };
			case 'Tab': return { name: 'Tab', rune: '\t' };
			case 'Escape': return { name: 'Escape' };
			case ' ': return { name: 'Space', rune: ' ' };
			default: {
				// F-keys
				const m = /^F([1-9]|1[0-9]|2[0-4])$/.exec(key);
				if (m) {
					return { name: `F${m[1]}` };
				}
				return null;
			}
		}
	}
}
