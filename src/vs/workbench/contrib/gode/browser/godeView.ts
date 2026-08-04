/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { View, IContentWidgetData, IOverlayWidgetData, IGlyphMarginWidgetData } from '../../../../editor/browser/view.js';
import { ICommandDelegate } from '../../../../editor/browser/view/viewController.js';
import { ViewUserInputEvents } from '../../../../editor/browser/view/viewUserInputEvents.js';
import { OverviewRuler } from '../../../../editor/browser/viewParts/overviewRuler/overviewRuler.js';
import { IEditorAriaOptions, IMouseTarget } from '../../../../editor/browser/editorBrowser.js';
import { IEditorConfiguration } from '../../../../editor/common/config/editorConfiguration.js';
import { EditorOption } from '../../../../editor/common/config/editorOptions.js';
import { IColorTheme, IThemeService } from '../../../../platform/theme/common/themeService.js';
import { IViewModel } from '../../../../editor/common/viewModel.js';
import { ViewEvent } from '../../../../editor/common/viewEvents.js';
import * as viewEvents from '../../../../editor/common/viewEvents.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IUserInteractionService } from '../../../../platform/userInteraction/browser/userInteractionService.js';
import { Color } from '../../../../base/common/color.js';
import { TokenizationRegistry } from '../../../../editor/common/languages.js';
import { GodeEngineClient } from './godeEngineClient.js';
import { GODE_ENGINE_PORT, ITokenLine } from '../common/godeProtocol.js';

/**
 * GodeView replaces the default DOM-based editor view with a canvas that
 * displays frames rendered by the gogpu/ui-based gode-engine process.
 *
 * The editor's text model stays in VS Code; the Go engine holds a mirror and
 * renders it offscreen. Edits made in the Go engine are replayed back into the
 * VS Code model via the command delegate.
 */
export class GodeView extends View {

	private readonly _client: GodeEngineClient;
	private readonly _canvas: HTMLCanvasElement;
	private readonly _canvasCtx: CanvasRenderingContext2D | null;
	private _disposed = false;

	// Re-entrancy / sync bookkeeping.
	private _applyingEngineEdit = false; // engine -> VS Code pushEditOperations in flight
	private _applyingSelection = false;  // engine <-> VS Code selection sync loop guard

	constructor(
		editorContainer: HTMLElement,
		ownerID: string,
		commandDelegate: ICommandDelegate,
		configuration: IEditorConfiguration,
		colorTheme: IColorTheme,
		model: IViewModel,
		userInputEvents: ViewUserInputEvents,
		overflowWidgetsDomNode: HTMLElement | undefined,
		@IInstantiationService instantiationService: IInstantiationService,
		@IUserInteractionService userInteractionService: IUserInteractionService,
	) {
		super(editorContainer, ownerID, commandDelegate, configuration, colorTheme, model, userInputEvents, overflowWidgetsDomNode, instantiationService, userInteractionService);

		// Keep the default view's DOM structure (glyph margin, line numbers, etc.)
		// but overlay our canvas on top. This preserves VS Code's breakpoint and
		// decoration infrastructure while using the Go engine for text rendering.
		const domNode = this.domNode.domNode;
		domNode.style.backgroundColor = '#1e1e1e';

		this._canvas = document.createElement('canvas');
		this._canvas.style.position = 'absolute';
		this._canvas.style.inset = '0';
		this._canvas.style.width = '100%';
		this._canvas.style.height = '100%';
		this._canvas.style.pointerEvents = 'none'; // Let clicks pass through to DOM
		this._canvas.setAttribute('tabindex', '0');
		domNode.appendChild(this._canvas);

		const ctx = this._canvas.getContext('2d');
		if (!ctx) {
			throw new Error('GodeView: cannot create 2d canvas context');
		}
		this._canvasCtx = ctx;

		this._client = new GodeEngineClient(GODE_ENGINE_PORT, this._canvas, this._canvasCtx);

		// --- Wire the engine's edit events back into the VS Code model ---
		// Guarded so the resulting VS Code content change is not re-synced back
		// to the engine (which would echo and diverge).
		this._client.onDidEdit((range, text) => {
			const model = this._viewModel.model;
			if (!range) {
				return;
			}
			const start = model.validatePosition({ lineNumber: range.start.line, column: range.start.column });
			const end = model.validatePosition({ lineNumber: range.end.line, column: range.end.column });
			this._applyingEngineEdit = true;
			try {
				model.pushEditOperations([], [{ range: { startLineNumber: start.lineNumber, startColumn: start.column, endLineNumber: end.lineNumber, endColumn: end.column }, text, forceMoveMarkers: false }], () => []);
			} finally {
				this._applyingEngineEdit = false;
			}
		});

		// --- Sync engine selection into VS Code (guarded against loops) ---
		this._client.onDidChangeSelection((anchor, active) => {
			if (this._applyingSelection) {
				return;
			}
			this._applyingSelection = true;
			try {
				// ViewModel.setSelections signature is (source, selections, reason);
				// omitting source made 'selections' the first arg and left the real
				// selections undefined (cursorCommon 'length' TypeError loop).
				(this._viewModel as any).setSelections('gode', [{ selectionStartLineNumber: anchor.line, selectionStartColumn: anchor.column, positionLineNumber: active.line, positionColumn: active.column }]);
			} finally {
				this._applyingSelection = false;
			}
		});

		// --- Sync VS Code model changes back into the engine mirror ---
		// Anything that edits the model outside the engine path (undo/redo,
		// find&replace, external edits, multi-cursor) would otherwise leave the
		// engine's mirror stale and subsequent edits would land at wrong spots.
		// Named textModel to avoid shadowing the IViewModel constructor parameter.
		const textModel = this._viewModel.model;
		this._register(textModel.onDidChangeContent(() => {
			if (this._applyingEngineEdit || this._disposed) {
				return;
			}
			this._client.setText(textModel.getValue());
			// Keep the engine caret on the same (VS Code) position. getSelection()
			// lives on the ViewModel impl but not the IViewModel interface, hence
			// the cast (same pattern as the selection-sync handler below).
			const sel = (this._viewModel as any).getSelection();
			this._client.setSelection(
				{ line: sel.selectionStartLineNumber, column: sel.selectionStartColumn },
				{ line: sel.positionLineNumber, column: sel.positionColumn }
			);
			// The engine replaced its model; refresh token colors for the new text.
			this._syncAllTokens();
		}));

		// --- Sync VS Code tokenization colors into the engine ---
		// The engine owns scroll, so we cannot rely on VS Code's visible range.
		// Instead: send all lines once up front, then send each changed range as
		// VS Code (re)tokenizes, and re-send everything when the theme changes.
		this._register(textModel.onDidChangeTokens((e: { ranges: { fromLineNumber: number; toLineNumber: number }[] }) => {
			for (const r of e.ranges) {
				this._sendTokensForRange(r.fromLineNumber, r.toLineNumber);
			}
		}));
		const themeService = instantiationService.invokeFunction(accessor => accessor.get(IThemeService));
		this._register(themeService.onDidColorThemeChange(() => {
			this._syncAllTokens();
		}));

		// --- Sync VS Code breakpoint decorations into the engine ---
		// Breakpoints are rendered as glyph-margin decorations by the debug
		// contribution. The engine draws its own marker in the glyph margin, so we
		// mirror the set of lines carrying a breakpoint decoration.
		this._register(textModel.onDidChangeDecorations(() => {
			this._syncBreakpoints();
		}));

		// --- Forward input events from the DOM canvas to the engine ---
		this._installDomListeners();

		// Open the current document in the engine.
		this._syncDocument();
		this._syncAllTokens();
		this._syncBreakpoints();
		this._syncGlyphMarginWidth();
	}

	// --- Event handling: bypass the default view part rendering loop ---

	public override handleEvents(events: ViewEvent[]): void {
		for (const e of events) {
			if (e.type === viewEvents.ViewEventType.ViewCursorStateChanged) {
				this.onCursorStateChanged(e as viewEvents.ViewCursorStateChangedEvent);
			}
		}
	}

	private get _viewModel(): IViewModel {
		return (this as any)._context.viewModel as IViewModel;
	}

	// --- Document sync ---

	private _syncDocument(): void {
		const model = this._viewModel.model;
		const text = model.getValue();
		this._client.openDocument(text);
	}

	// --- Breakpoint / glyph margin sync ---

	/**
	 * Send the glyph-margin gutter width (in device pixels) to the engine so its
	 * text area aligns with VS Code's breakpoint gutter. Re-sent on every render
	 * so font/layout changes (zoom, theme) stay in sync.
	 */
	private _syncGlyphMarginWidth(): void {
		const opts = (this as any)._context.configuration.options;
		const layoutInfo = opts.get(EditorOption.layoutInfo);
		if (!layoutInfo) {
			return;
		}
		const dpr = window.devicePixelRatio || 1;
		const cssWidth = layoutInfo.glyphMarginLeft + layoutInfo.glyphMarginWidth;
		this._client.setGlyphMarginWidth(Math.ceil(cssWidth * dpr));
	}

	/**
	 * Mirror the set of lines carrying a breakpoint glyph-margin decoration into
	 * the engine, which renders the breakpoint dot. Lines are 1-based.
	 */
	private _syncBreakpoints(): void {
		if (this._disposed) {
			return;
		}
		const model = this._viewModel.model;
		const lines: number[] = [];
		for (const dec of model.getAllDecorations()) {
			const cls = dec.options.glyphMarginClassName;
			if (cls && cls.includes('breakpoint')) {
				lines.push(dec.range.startLineNumber);
			}
		}
		this._client.setBreakpoints(lines);
	}

	// --- Rendering ---

	public override render(now: boolean, everything: boolean): void {
		// The engine renders asynchronously; the canvas is updated on each frame.
		// The engine works in device pixels: pass the physical size and the
		// device pixel ratio so fonts/lines scale 1:1 with the display and the
		// frame is not upscaled (blurry) on HiDPI screens.
		const size = this.domNode.domNode.getBoundingClientRect();
		if (size.width > 0 && size.height > 0) {
			const dpr = window.devicePixelRatio || 1;
			this._client.setViewport(Math.ceil(size.width * dpr), Math.ceil(size.height * dpr), dpr);
			// Keep the breakpoint gutter width in sync with layout/zoom changes.
			this._syncGlyphMarginWidth();
		}
		this._client.requestFrame();
	}

	// --- Focus ---

	public override focus(): void {
		this._canvas.focus();
		this._canvas.setAttribute('tabindex', '0');
		// Tell the engine to focus its EditorView too. Without this the engine
		// drops key events (handleKey requires IsFocused()) when the editor is
		// focused programmatically (tab switch, etc.), making it feel read-only.
		this._client.focus();
	}

	public override isFocused(): boolean {
		return document.activeElement === this._canvas;
	}

	public override isWidgetFocused(): boolean {
		return this.isFocused();
	}

	public override refreshFocusState(): void {
		// no-op: canvas focus is tracked directly.
	}

	// --- Selection from VS Code (e.g. commands, find) ---

	public override onCursorStateChanged(e: viewEvents.ViewCursorStateChangedEvent): boolean {
		if (this._applyingSelection) {
			return false;
		}
		const sel = e.selections[0];
		if (sel) {
			this._client.setSelection(
				{ line: sel.selectionStartLineNumber, column: sel.selectionStartColumn },
				{ line: sel.positionLineNumber, column: sel.positionColumn }
			);
		}
		return false;
	}

	// --- View helpers that the engine replaces ---

	public override getOffsetForColumn(modelLineNumber: number, modelColumn: number): number {
		// Approximate: offset = textLeft + (column-1)*charWidth. The engine owns
		// exact metrics; this is used for widget anchoring and is good enough.
		const opts = (this as any)._context.configuration.options as any;
		const fontSize = opts.get(EditorOption.fontSize);
		const charWidth = fontSize * 0.6;
		return Math.round((modelColumn - 1) * charWidth);
	}

	public override getLineWidth(modelLineNumber: number): number {
		const model = this._viewModel.model;
		const text = model.getLineContent(modelLineNumber);
		const opts = (this as any)._context.configuration.options as any;
		const fontSize = opts.get(EditorOption.fontSize);
		return Math.round(text.length * fontSize * 0.6);
	}

	public override resetLineWidthCaches(): void {
		// no-op
	}

	public override getTargetAtClientPoint(clientX: number, clientY: number): IMouseTarget | null {
		return null;
	}

	public override createOverviewRuler(cssClassName: string): OverviewRuler {
		return new OverviewRuler((this as any)._context, cssClassName);
	}

	public override change(callback: (changeAccessor: any) => unknown): void {
		// View zones are not supported by the engine; ignore.
	}

	public override writeScreenReaderContent(reason: string): void {
		// no-op
	}

	public override setAriaOptions(options: IEditorAriaOptions): void {
		// no-op
	}

	public override addContentWidget(widgetData: IContentWidgetData): void {
		// no-op
	}

	public override layoutContentWidget(widgetData: IContentWidgetData): void {
		// no-op
	}

	public override removeContentWidget(widgetData: IContentWidgetData): void {
		// no-op
	}

	public override addOverlayWidget(widgetData: IOverlayWidgetData): void {
		// no-op
	}

	public override layoutOverlayWidget(widgetData: IOverlayWidgetData): void {
		// no-op
	}

	public override removeOverlayWidget(widgetData: IOverlayWidgetData): void {
		// no-op
	}

	public override addGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// no-op
	}

	public override layoutGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// no-op
	}

	public override removeGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// no-op
	}

	public override dispose(): void {
		if (this._disposed) {
			return;
		}
		this._disposed = true;
		this._client.dispose();
		super.dispose();
	}

	// --- Input forwarding ---
	//
	// Only "editing" keys are intercepted and forwarded to the Go engine.
	// Everything else (Ctrl/Cmd+S, Ctrl/Cmd+C/V, command palette, ...) is left
	// for VS Code's global shortcut handling. The engine keeps the VS Code
	// model and selection in sync, so VS Code commands keep working.

	private _installDomListeners(): void {
		const domNode = this.domNode.domNode;

		domNode.addEventListener('keydown', (e: KeyboardEvent) => {
			if (this._isEditingKey(e)) {
				e.preventDefault();
				e.stopImmediatePropagation();
				this._client.sendKey('press', e);
			}
		}, true);

		domNode.addEventListener('mousedown', (e: MouseEvent) => {
			if (e.button === 2) {
				return; // right-click: keep VS Code context menu
			}
			// Clicks in the glyph margin (breakpoint gutter) are handed to VS
			// Code so the debug contribution can toggle breakpoints there. The
			// engine does not need them. Everything else goes to the engine.
			if (this._isInGlyphMargin(e)) {
				return;
			}
			e.preventDefault();
			e.stopImmediatePropagation();
			this._canvas.focus();
			this._client.sendMouse('press', e);
		}, true);

		domNode.addEventListener('mousemove', (e: MouseEvent) => {
			// The engine's press handler sets its internal dragging flag; while
			// the left button is held, moves must be reported as 'drag' so the
			// engine extends the selection. Plain 'move' events carry no button
			// state to the engine and would be ignored during a drag.
			this._client.sendMouse((e.buttons & 1) ? 'drag' : 'move', e);
		}, true);

		domNode.addEventListener('mouseup', (e: MouseEvent) => {
			if (e.button === 2 || this._isInGlyphMargin(e)) {
				return;
			}
			e.preventDefault();
			e.stopImmediatePropagation();
			this._client.sendMouse('release', e);
		}, true);

		domNode.addEventListener('wheel', (e: WheelEvent) => {
			e.preventDefault();
			e.stopImmediatePropagation();
			const [dx, dy] = this._normalizeWheelDelta(e);
			// The engine scrolls in device pixels, so scale the CSS-pixel delta.
			const dpr = window.devicePixelRatio || 1;
			this._client.sendWheelDelta(dx * dpr, dy * dpr);
		}, true);
	}

	/**
	 * True when the click falls inside VS Code's glyph margin (breakpoint
	 * gutter) in CSS pixels. Those clicks must reach VS Code's own mouse
	 * handling so the debug contribution can toggle breakpoints.
	 */
	private _isInGlyphMargin(e: MouseEvent): boolean {
		const rect = this._canvas.getBoundingClientRect();
		const x = e.clientX - rect.left;
		const opts = (this as any)._context.configuration.options;
		const layoutInfo = opts.get(EditorOption.layoutInfo);
		if (!layoutInfo) {
			return false;
		}
		return x >= layoutInfo.glyphMarginLeft && x < layoutInfo.glyphMarginLeft + layoutInfo.glyphMarginWidth;
	}

	/**
	 * Decides whether a key event is an "editing" key that the Go engine owns.
	 * Navigation (arrows/Home/End/PageUp/PageDown, optionally with Shift for
	 * selection or Ctrl/Cmd for word/document moves) and plain text input are
	 * forwarded; everything else stays in VS Code.
	 */
	private _isEditingKey(e: KeyboardEvent): boolean {
		const navKeys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End', 'PageUp', 'PageDown'];
		const editKeys = ['Enter', 'Backspace', 'Delete', 'Tab', ' '];

		if (navKeys.includes(e.key)) {
			// Only navigation alone or with Shift / Ctrl / Cmd (not Alt).
			return !e.altKey;
		}
		if (editKeys.includes(e.key)) {
			return !e.ctrlKey && !e.metaKey && !e.altKey;
		}
		if (e.key.length === 1) {
			// Printable characters: only plain (no Ctrl/Cmd/Alt) input.
			return !e.ctrlKey && !e.metaKey && !e.altKey;
		}
		return false;
	}

	// --- Wheel normalization ---

	/**
	 * Normalize a wheel event to pixel deltas, applying VS Code's
	 * mouseWheelScrollSensitivity and fastScrollSensitivity (Alt) multipliers
	 * and converting line/page delta modes to pixels. The engine then scrolls
	 * by raw pixels, which matches VS Code's feel instead of the previous
	 * jumpy/too-slow scrolling caused by forwarding raw deltas.
	 */
	private _normalizeWheelDelta(e: WheelEvent): [number, number] {
		const options = (this as any)._context.configuration.options;
		const sensitivity = options.get(EditorOption.mouseWheelScrollSensitivity) ?? 1;
		const fastSensitivity = options.get(EditorOption.fastScrollSensitivity) ?? 5;
		const mult = sensitivity * (e.altKey ? fastSensitivity : 1);

		let dy: number;
		let dx: number;
		if (e.deltaMode === WheelEvent.DOM_DELTA_LINE) {
			const lineHeight = options.get(EditorOption.lineHeight);
			const fontSize = options.get(EditorOption.fontSize);
			dy = e.deltaY * lineHeight * mult;
			dx = e.deltaX * fontSize * mult;
		} else if (e.deltaMode === WheelEvent.DOM_DELTA_PAGE) {
			const layoutInfo = options.get(EditorOption.layoutInfo);
			dy = e.deltaY * (layoutInfo?.height ?? 600) * mult;
			dx = e.deltaX * (layoutInfo?.width ?? 800) * mult;
		} else {
			dy = e.deltaY * mult;
			dx = e.deltaX * mult;
		}
		return [dx, dy];
	}

	// --- Syntax highlighting (VS Code tokenization -> engine) ---

	/** Send token colors for every line (initial open + theme change + resync). */
	private _syncAllTokens(): void {
		if (this._disposed) {
			return;
		}
		const lineCount = this._viewModel.model.getLineCount();
		if (lineCount === 0) {
			return;
		}
		this._sendTokensForRange(1, lineCount);
	}

	private _sendTokensForRange(startLine: number, endLine: number): void {
		if (this._disposed) {
			return;
		}
		const model = this._viewModel.model;
		const lineCount = model.getLineCount();
		if (startLine < 1) {
			startLine = 1;
		}
		if (endLine > lineCount) {
			endLine = lineCount;
		}
		if (startLine > endLine) {
			return;
		}
		const colorMap = TokenizationRegistry.getColorMap();
		const lines: ITokenLine[] = [];
		for (let line = startLine; line <= endLine; line++) {
			const spans = GodeView._buildTokenSpans(model, line, colorMap);
			if (spans.length > 0) {
				lines.push({ line, spans });
			}
		}
		if (lines.length > 0) {
			this._client.setTokens(lines);
		}
	}

	private static _buildTokenSpans(model: { tokenization: { getLineTokens(lineNumber: number): { getCount(): number; getEndOffset(i: number): number; getForeground(i: number): number } } }, line: number, colorMap: readonly Color[] | null): { start: number; end: number; color: string }[] {
		const lineTokens = model.tokenization.getLineTokens(line);
		const count = lineTokens.getCount();
		const spans: { start: number; end: number; color: string }[] = [];
		let prevEnd = 0; // 0-based, exclusive offset of the previous token's end
		for (let i = 0; i < count; i++) {
			const endOffset = lineTokens.getEndOffset(i); // 0-based, exclusive
			const fg = lineTokens.getForeground(i);
			const color = (colorMap && colorMap[fg]) ? Color.Format.CSS.formatHex(colorMap[fg]) : '';
			// Convert 0-based offsets [prevEnd, endOffset) to 1-based columns
			// [prevEnd+1, endOffset+1).
			const startCol = prevEnd + 1;
			const endCol = endOffset + 1;
			if (endCol > startCol) {
				spans.push({ start: startCol, end: endCol, color });
			}
			prevEnd = endOffset;
		}
		return spans;
	}
}
