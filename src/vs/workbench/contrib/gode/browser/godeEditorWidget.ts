/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Widget } from '../../../../base/browser/ui/widget.js';
import { Emitter } from '../../../../base/common/event.js';
import { ILogService } from '../../../../platform/log/common/log.js';
import { GodeRenderer, IGodeRenderer } from './godeRenderer.js';
import { IPos, IRange } from '../common/godeProtocol.js';

export interface IGodeEditorWidgetOptions {
	fontSize?: number;
	lineHeight?: number;
	tabSize?: number;
	lineNumbers?: boolean;
}

export class GodeEditorWidget extends Widget {

	readonly domNode: HTMLElement;
	private canvas: HTMLCanvasElement;
	private renderer: IGodeRenderer;

	private readonly _onDidChangeSelection = this._register(new Emitter<{ anchor: IPos; active: IPos }>());
	readonly onDidChangeSelection = this._onDidChangeSelection.event;

	private readonly _onDidChangeContent = this._register(new Emitter<{ range: IRange; text: string }>());
	readonly onDidChangeContent = this._onDidChangeContent.event;

	private readonly _onDidFocusEditorText = this._register(new Emitter<void>());
	readonly onDidFocusEditorText = this._onDidFocusEditorText.event;

	private readonly _onDidBlurEditorText = this._register(new Emitter<void>());
	readonly onDidBlurEditorText = this._onDidBlurEditorText.event;

	private hasFocus = false;
	private lastSelection: { anchor: IPos; active: IPos } = { anchor: { line: 1, column: 1 }, active: { line: 1, column: 1 } };

	constructor(
		container: HTMLElement,
		@ILogService private readonly logService: ILogService,
		_options: IGodeEditorWidgetOptions = {}
	) {
		super();

		this.domNode = document.createElement('div');
		this.domNode.className = 'gode-editor-widget';
		this.domNode.style.position = 'relative';
		this.domNode.style.overflow = 'hidden';
		this.domNode.style.outline = 'none';
		this.domNode.tabIndex = 0;

		// Create canvas for rendering
		this.canvas = document.createElement('canvas');
		this.canvas.style.position = 'absolute';
		this.canvas.style.top = '0';
		this.canvas.style.left = '0';
		this.canvas.style.width = '100%';
		this.canvas.style.height = '100%';
		this.canvas.style.display = 'block';
		this.domNode.appendChild(this.canvas);

		// Create renderer
		this.renderer = new GodeRenderer(this.logService);

		// Register renderer events
		this._register(this.renderer.onSelectionChanged((sel) => {
			this.lastSelection = sel;
			this._onDidChangeSelection.fire(sel);
		}));

		this._register(this.renderer.onEdited((edit) => {
			this._onDidChangeContent.fire({ range: edit.range, text: edit.editText });
		}));

		this._register(this.renderer.onReady(() => {
			this.logService.info('[gode] Editor widget connected to engine');
			this.renderer.renderTo(this.canvas);
			this.updateSize();
		}));

		this._register(this.renderer.onError((err) => {
			this.logService.error(`[gode] Renderer error: ${err.message}`);
		}));

		// Setup input handling
		this.setupInputHandling();

		// Setup focus handling
		this.setupFocusHandling();

		// Append to container
		container.appendChild(this.domNode);

		// Setup resize observer
		const resizeObserver = new ResizeObserver(() => {
			this.updateSize();
		});
		resizeObserver.observe(this.domNode);
		this._register({ dispose: () => resizeObserver.disconnect() });

		// Initial size update
		this.updateSize();
	}

	private setupInputHandling(): void {
		// Keyboard handling
		this.domNode.addEventListener('keydown', (e) => {
			if (!this.hasFocus) {
				this.domNode.focus();
			}
			e.preventDefault();

			const keyEvent = this.toKeyEvent(e);
			this.renderer.sendKeyEvent(keyEvent);
		});

		this.domNode.addEventListener('keyup', (e) => {
			const keyEvent = this.toKeyEvent(e);
			keyEvent.key_type = 'release';
			this.renderer.sendKeyEvent(keyEvent);
		});

		// Mouse handling
		this.canvas.addEventListener('mousedown', (e) => {
			this.domNode.focus();
			const mouseEvent = this.toMouseEvent(e, 'press');
			this.renderer.sendMouseEvent(mouseEvent);
		});

		this.canvas.addEventListener('mousemove', (e) => {
			if (e.buttons === 1) { // Left button pressed
				const mouseEvent = this.toMouseEvent(e, 'drag');
				this.renderer.sendMouseEvent(mouseEvent);
			} else {
				const mouseEvent = this.toMouseEvent(e, 'move');
				this.renderer.sendMouseEvent(mouseEvent);
			}
		});

		this.canvas.addEventListener('mouseup', (e) => {
			const mouseEvent = this.toMouseEvent(e, 'release');
			this.renderer.sendMouseEvent(mouseEvent);
		});

		this.canvas.addEventListener('dblclick', (e) => {
			const mouseEvent = this.toMouseEvent(e, 'double_click');
			this.renderer.sendMouseEvent(mouseEvent);
		});

		// Wheel handling
		this.canvas.addEventListener('wheel', (e) => {
			e.preventDefault();
			const wheelEvent = {
				dx: 0,
				dy: -e.deltaY,
				shift: e.shiftKey,
				ctrl: e.ctrlKey
			};
			this.renderer.sendWheelEvent(wheelEvent);
		}, { passive: false });

		// Context menu prevention (for right-click selection)
		this.canvas.addEventListener('contextmenu', (e) => {
			e.preventDefault();
			const mouseEvent = this.toMouseEvent(e, 'press');
			mouseEvent.button = 'right';
			this.renderer.sendMouseEvent(mouseEvent);
		});

		// Paste handling
		this.domNode.addEventListener('paste', async (e) => {
			e.preventDefault();
			const text = await navigator.clipboard.readText();
			// Send text as key events character by character
			for (const char of text) {
				const keyEvent = {
					key_type: 'press' as const,
					key: this.charToKey(char),
					rune: char,
					shift: false,
					ctrl: false,
					alt: false,
					super: false
				};
				this.renderer.sendKeyEvent(keyEvent);
			}
		});

		// Copy handling
		this.domNode.addEventListener('copy', async (e) => {
			e.preventDefault();
			try {
				const content = await this.renderer.getContent(0);
				await navigator.clipboard.writeText(content);
			} catch (err) {
				this.logService.error(`[gode] Copy failed: ${err}`);
			}
		});

		// Cut handling
		this.domNode.addEventListener('cut', async (e) => {
			e.preventDefault();
			try {
				const content = await this.renderer.getContent(0);
				await navigator.clipboard.writeText(content);
				// Send delete key to remove selection
				this.renderer.sendKeyEvent({
					key_type: 'press',
					key: 'Delete',
					rune: '',
					shift: false,
					ctrl: false,
					alt: false,
					super: false
				});
			} catch (err) {
				this.logService.error(`[gode] Cut failed: ${err}`);
			}
		});
	}

	private setupFocusHandling(): void {
		this.domNode.addEventListener('focus', () => {
			this.hasFocus = true;
			this._onDidFocusEditorText.fire();
		});

		this.domNode.addEventListener('blur', () => {
			this.hasFocus = false;
			this._onDidBlurEditorText.fire();
		});

		// Handle focus when clicking on canvas
		this.canvas.addEventListener('mousedown', () => {
			this.domNode.focus();
		});
	}

	private toKeyEvent(e: KeyboardEvent): {
		key_type: 'press' | 'release' | 'repeat';
		key: string;
		rune: string;
		shift: boolean;
		ctrl: boolean;
		alt: boolean;
		super: boolean;
	} {
		let key = e.key;
		let rune = '';

		// Map special keys
		switch (e.key) {
			case 'ArrowUp':
				key = 'Up';
				break;
			case 'ArrowDown':
				key = 'Down';
				break;
			case 'ArrowLeft':
				key = 'Left';
				break;
			case 'ArrowRight':
				key = 'Right';
				break;
			case 'Enter':
				key = 'Enter';
				break;
			case 'Backspace':
				key = 'Backspace';
				break;
			case 'Delete':
				key = 'Delete';
				break;
			case 'Tab':
				key = 'Tab';
				break;
			case 'Escape':
				key = 'Escape';
				break;
			case 'Home':
				key = 'Home';
				break;
			case 'End':
				key = 'End';
				break;
			case 'PageUp':
				key = 'PageUp';
				break;
			case 'PageDown':
				key = 'PageDown';
				break;
			default:
				// Printable characters
				if (e.key.length === 1) {
					rune = e.key;
					key = e.key.toUpperCase();
				}
				break;
		}

		return {
			key_type: e.repeat ? 'repeat' : 'press',
			key,
			rune,
			shift: e.shiftKey,
			ctrl: e.ctrlKey,
			alt: e.altKey,
			super: e.metaKey || e.key === 'Meta'
		};
	}

	private toMouseEvent(e: MouseEvent, type: 'press' | 'release' | 'move' | 'drag' | 'double_click'): {
		mouse_type: 'press' | 'release' | 'move' | 'drag' | 'double_click';
		button: 'left' | 'right' | 'middle';
		x: number;
		y: number;
		shift: boolean;
		ctrl: boolean;
		alt: boolean;
		super: boolean;
	} {
		const rect = this.canvas.getBoundingClientRect();
		const scaleX = this.canvas.width / rect.width;
		const scaleY = this.canvas.height / rect.height;

		return {
			mouse_type: type,
			button: e.button === 0 ? 'left' : e.button === 1 ? 'middle' : 'right',
			x: (e.clientX - rect.left) * scaleX,
			y: (e.clientY - rect.top) * scaleY,
			shift: e.shiftKey,
			ctrl: e.ctrlKey,
			alt: e.altKey,
			super: e.metaKey
		};
	}

	private charToKey(char: string): string {
		const upper = char.toUpperCase();
		if (upper >= 'A' && upper <= 'Z') return upper;
		if (char >= '0' && char <= '9') return char;
		return char;
	}

	private updateSize(): void {
		const rect = this.domNode.getBoundingClientRect();
		if (rect.width > 0 && rect.height > 0) {
			const dpr = window.devicePixelRatio || 1;
			const width = Math.floor(rect.width * dpr);
			const height = Math.floor(rect.height * dpr);

			// Resize canvas
			this.canvas.width = width;
			this.canvas.height = height;
			this.canvas.style.width = `${rect.width}px`;
			this.canvas.style.height = `${rect.height}px`;

			// Update engine viewport
			this.renderer.setViewport(width, height);
		}
	}

	// --- Public API ---

	async connect(): Promise<void> {
		await this.renderer.connect();
	}

	setText(text: string): void {
		this.renderer.setText(text);
	}

	getSelection(): { anchor: IPos; active: IPos } {
		return this.lastSelection;
	}

	setSelection(anchor: IPos, active: IPos): void {
		this.renderer.setSelection(anchor, active);
	}

	focus(): void {
		this.domNode.focus();
	}

	isFocused(): boolean {
		return this.hasFocus;
	}

	layout(): void {
		this.updateSize();
	}

	override dispose(): void {
		this.renderer.disconnect();
		super.dispose();
	}
}
