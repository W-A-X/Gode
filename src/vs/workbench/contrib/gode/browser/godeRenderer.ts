/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable } from '../../../../base/common/lifecycle.js';
import { Emitter, Event } from '../../../../base/common/event.js';
import { ILogService } from '../../../../platform/log/common/log.js';
import { IGodeCommand, IGodeEvent, IPos, IRange } from '../common/godeProtocol.js';

const GODE_ENGINE_PORT = 47810;

export interface IGodeRenderer {
	readonly onSelectionChanged: Event<{ anchor: IPos; active: IPos }>;
	readonly onEdited: Event<{ range: IRange; editText: string }>;
	readonly onReady: Event<void>;
	readonly onError: Event<Error>;

	connect(): Promise<void>;
	disconnect(): void;
	isConnected(): boolean;

	sendCommand(cmd: IGodeCommand): void;
	setText(text: string): void;
	setViewport(width: number, height: number): void;
	setSelection(anchor: IPos, active: IPos): void;
	sendKeyEvent(key: {
		key_type: 'press' | 'release' | 'repeat';
		key: string;
		rune: string;
		shift?: boolean;
		ctrl?: boolean;
		alt?: boolean;
		super?: boolean;
	}): void;
	sendMouseEvent(mouse: {
		mouse_type: 'press' | 'release' | 'move' | 'drag' | 'double_click';
		button: 'left' | 'right' | 'middle';
		x: number;
		y: number;
		shift?: boolean;
		ctrl?: boolean;
		alt?: boolean;
		super?: boolean;
	}): void;
	sendWheelEvent(wheel: {
		dx: number;
		dy: number;
		shift?: boolean;
		ctrl?: boolean;
	}): void;
	getContent(id: number): Promise<string>;

	renderTo(canvas: HTMLCanvasElement): void;
}

export class GodeRenderer extends Disposable implements IGodeRenderer {

	private ws: WebSocket | null = null;
	private canvas: HTMLCanvasElement | null = null;
	private ctx: CanvasRenderingContext2D | null = null;
	private currentFrameData: ImageData | null = null;

	private readonly _onSelectionChanged = this._register(new Emitter<{ anchor: IPos; active: IPos }>());
	readonly onSelectionChanged = this._onSelectionChanged.event;

	private readonly _onEdited = this._register(new Emitter<{ range: IRange; editText: string }>());
	readonly onEdited = this._onEdited.event;

	private readonly _onReady = this._register(new Emitter<void>());
	readonly onReady = this._onReady.event;

	private readonly _onError = this._register(new Emitter<Error>());
	readonly onError = this._onError.event;

	private pendingContentRequests = new Map<number, { resolve: (value: string) => void; reject: (reason: unknown) => void }>();

	constructor(
		@ILogService private readonly logService: ILogService,
		private readonly port: number = GODE_ENGINE_PORT
	) {
		super();
	}

	async connect(): Promise<void> {
		if (this.ws?.readyState === WebSocket.OPEN) {
			return;
		}

		return new Promise<void>((resolve, reject) => {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const host = window.location.hostname || '127.0.0.1';
			const url = `${protocol}//${host}:${this.port}`;

			try {
				this.ws = new WebSocket(url);
				this.ws.binaryType = 'arraybuffer';

				const timeout = setTimeout(() => {
					reject(new Error('Connection timeout'));
					this.ws?.close();
					this.ws = null;
				}, 5000);

				this.ws.onopen = () => {
					clearTimeout(timeout);
					this.logService.info(`[gode] Connected to engine at ${url}`);
					resolve();
				};

				this.ws.onmessage = (event: MessageEvent) => {
					this.handleMessage(event);
				};

				this.ws.onerror = () => {
					clearTimeout(timeout);
					const error = new Error(`WebSocket connection error`);
					this._onError.fire(error);
					reject(error);
				};

				this.ws.onclose = () => {
					this.logService.info('[gode] Disconnected from engine');
					this.ws = null;
				};
			} catch (err) {
				reject(err);
			}
		});
	}

	disconnect(): void {
		if (this.ws) {
			this.sendRaw({ cmd: 'shutdown' });
			this.ws.close();
			this.ws = null;
		}
	}

	isConnected(): boolean {
		return this.ws?.readyState === WebSocket.OPEN;
	}

	sendCommand(cmd: IGodeCommand): void {
		this.sendRaw(cmd);
	}

	setText(text: string): void {
		this.sendRaw({ cmd: 'open_document', text });
	}

	setViewport(width: number, height: number): void {
		this.sendRaw({ cmd: 'set_viewport', width, height });
	}

	setSelection(anchor: IPos, active: IPos): void {
		this.sendRaw({ cmd: 'set_selection', anchor, active });
	}

	sendKeyEvent(key: IGodeCommand['key'] & object): void {
		this.sendRaw({ cmd: 'input', type: 'key', key });
	}

	sendMouseEvent(mouse: IGodeCommand['mouse'] & object): void {
		this.sendRaw({ cmd: 'input', type: 'mouse', mouse });
	}

	sendWheelEvent(wheel: IGodeCommand['wheel'] & object): void {
		this.sendRaw({ cmd: 'input', type: 'wheel', wheel });
	}

	async getContent(id: number): Promise<string> {
		return new Promise<string>((resolve, reject) => {
			this.pendingContentRequests.set(id, { resolve, reject });
			this.sendRaw({ cmd: 'get_content', id });

			setTimeout(() => {
				if (this.pendingContentRequests.has(id)) {
					this.pendingContentRequests.delete(id);
					reject(new Error('getContent timeout'));
				}
			}, 5000);
		});
	}

	renderTo(canvas: HTMLCanvasElement): void {
		this.canvas = canvas;
		this.ctx = canvas.getContext('2d');
		this.applyFrame();
	}

	private sendRaw(cmd: IGodeCommand): void {
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify(cmd));
		}
	}

	private handleMessage(event: MessageEvent): void {
		if (typeof event.data === 'string') {
			try {
				const msg: IGodeEvent = JSON.parse(event.data);
				this.handleEvent(msg);
			} catch (err) {
				this.logService.error(`[gode] Failed to parse message: ${err}`);
			}
		} else if (event.data instanceof ArrayBuffer) {
			this.handleBinaryData(event.data);
		}
	}

	private async handleEvent(msg: IGodeEvent): Promise<void> {
		switch (msg.evt) {
			case 'ready':
				this._onReady.fire();
				break;

			case 'frame':
				if (msg.data && typeof msg.data === 'string') {
					// Base64 encoded RGBA data
					const binary = this.base64ToArrayBuffer(msg.data);
					this.updateFrame(binary, msg.width || 0, msg.height || 0);
				}
				if (msg.anchor && msg.active) {
					this._onSelectionChanged.fire({ anchor: msg.anchor, active: msg.active });
				}
				break;

			case 'edited':
				if (msg.range && msg.edit_text !== undefined) {
					this._onEdited.fire({ range: msg.range, editText: msg.edit_text });
				}
				break;

			case 'content':
				if (msg.id !== undefined && this.pendingContentRequests.has(msg.id)) {
					const pending = this.pendingContentRequests.get(msg.id)!;
					this.pendingContentRequests.delete(msg.id);
					pending.resolve(msg.content || '');
				}
				break;

			case 'pong':
				break;
		}
	}

	private handleBinaryData(data: ArrayBuffer): void {
		// Binary frame data - would be processed here for optimization
		this.logService.debug(`[gode] Received binary frame: ${data.byteLength} bytes`);
	}

	private updateFrame(rgbaData: ArrayBuffer, width: number, height: number): void {
		if (this.canvas && this.ctx && width > 0 && height > 0) {
			// Resize canvas if needed
			if (this.canvas.width !== width || this.canvas.height !== height) {
				this.canvas.width = width;
				this.canvas.height = height;
			}

			// Create ImageData from RGBA data
			const uint8 = new Uint8ClampedArray(rgbaData);
			this.currentFrameData = new ImageData(uint8, width, height);
			this.applyFrame();
		}
	}

	private applyFrame(): void {
		if (this.canvas && this.ctx && this.currentFrameData) {
			this.ctx.putImageData(this.currentFrameData, 0, 0);
		}
	}

	private base64ToArrayBuffer(base64: string): ArrayBuffer {
		const binaryString = atob(base64);
		const len = binaryString.length;
		const bytes = new Uint8Array(len);
		for (let i = 0; i < len; i++) {
			bytes[i] = binaryString.charCodeAt(i);
		}
		return bytes.buffer;
	}

	override dispose(): void {
		this.disconnect();
		super.dispose();
	}
}
