/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import * as dom from '../../base/browser/dom.js';
import { FastDomNode, createFastDomNode } from '../../base/browser/fastDomNode.js';
import { IMouseWheelEvent } from '../../base/browser/mouseEvent.js';
import { inputLatency } from '../../base/browser/performance.js';
import { CodeWindow } from '../../base/browser/window.js';
import { BugIndicatingError, onUnexpectedError } from '../../base/common/errors.js';
import { Disposable, DisposableStore, IDisposable } from '../../base/common/lifecycle.js';
import { IPointerHandlerHelper } from './controller/mouseHandler.js';
import { PointerHandlerLastRenderData } from './controller/mouseTarget.js';
import { PointerHandler } from './controller/pointerHandler.js';
import { IContentWidget, IContentWidgetPosition, IEditorAriaOptions, IGlyphMarginWidget, IGlyphMarginWidgetPosition, IMouseTarget, IOverlayWidget, IOverlayWidgetPosition, IViewZoneChangeAccessor } from './editorBrowser.js';
import { HorizontalPosition, IViewLines, LineVisibleRanges, RenderingContext, RestrictedRenderingContext } from './view/renderingContext.js';
import { ICommandDelegate, ViewController } from './view/viewController.js';
import { PartFingerprint, PartFingerprints, ViewPart } from './view/viewPart.js';
import { ViewUserInputEvents } from './view/viewUserInputEvents.js';
import { ViewContentWidgets } from './viewParts/contentWidgets/contentWidgets.js';
import { ViewOverlayWidgets } from './viewParts/overlayWidgets/overlayWidgets.js';
import { OverviewRuler } from './viewParts/overviewRuler/overviewRuler.js';
import { ViewZones } from './viewParts/viewZones/viewZones.js';
import { IEditorConfiguration } from '../common/config/editorConfiguration.js';
import { EditorOption } from '../common/config/editorOptions.js';
import { Position } from '../common/core/position.js';
import { Range } from '../common/core/range.js';
import { Selection } from '../common/core/selection.js';
import { ScrollType } from '../common/editorCommon.js';
import { ViewEventHandler } from '../common/viewEventHandler.js';
import * as viewEvents from '../common/viewEvents.js';
import { ViewportData } from '../common/viewLayout/viewLinesViewportData.js';
import { IViewModel } from '../common/viewModel.js';
import { ViewContext } from '../common/viewModel/viewContext.js';
import { IInstantiationService } from '../../platform/instantiation/common/instantiation.js';
import { IColorTheme, getThemeTypeSelector } from '../../platform/theme/common/themeService.js';
import { ViewGpuContext } from './gpu/viewGpuContext.js';
import { AbstractEditContext } from './controller/editContext/editContext.js';
import { IClipboardCopyEvent, IClipboardPasteEvent } from './controller/editContext/clipboardUtils.js';
import { IVisibleRangeProvider, TextAreaEditContext } from './controller/editContext/textArea/textAreaEditContext.js';
import { NativeEditContext } from './controller/editContext/native/nativeEditContext.js';
import { AccessibilitySupport } from '../../platform/accessibility/common/accessibility.js';
import { Event, Emitter } from '../../base/common/event.js';
import { IUserInteractionService } from '../../platform/userInteraction/browser/userInteractionService.js';


export interface IContentWidgetData {
	widget: IContentWidget;
	position: IContentWidgetPosition | null;
}

export interface IOverlayWidgetData {
	widget: IOverlayWidget;
	position: IOverlayWidgetPosition | null;
}

export interface IGlyphMarginWidgetData {
	widget: IGlyphMarginWidget;
	position: IGlyphMarginWidgetPosition;
}

/**
 * The DOM text renderer (`ViewLines` / `ViewLinesGpu`) has been removed: text
 * rendering is owned by the Go engine (gode-engine). The remaining view parts
 * (view zones, content/overlay widgets) still receive a rendering context for
 * layout, but no DOM line is available for text measurement.
 */
class NullViewLines implements IViewLines {
	linesVisibleRangesForRange(range: Range, includeNewLines: boolean): LineVisibleRanges[] | null {
		return null;
	}

	visibleRangeForPosition(position: Position): HorizontalPosition | null {
		return null;
	}
}

export class View extends ViewEventHandler {

	private _widgetFocusTracker: CodeEditorWidgetFocusTracker;

	private readonly _context: ViewContext;
	private readonly _viewGpuContext?: ViewGpuContext;
	private _selections: Selection[];

	// These are parts, but we must do some API related calls on them, so we keep a reference
	private readonly _viewZones: ViewZones;
	private readonly _contentWidgets: ViewContentWidgets;
	private readonly _overlayWidgets: ViewOverlayWidgets;
	private readonly _viewParts: ViewPart[];
	private readonly _viewController: ViewController;

	private _editContextEnabled: boolean;
	private _accessibilitySupport: AccessibilitySupport;
	private _editContext: AbstractEditContext;
	private readonly _editContextClipboardListeners = new DisposableStore();
	private readonly _pointerHandler: PointerHandler;

	// Clipboard events relayed from editContext
	private readonly _onWillCopy = this._register(new Emitter<IClipboardCopyEvent>());
	public readonly onWillCopy: Event<IClipboardCopyEvent> = this._onWillCopy.event;

	private readonly _onWillCut = this._register(new Emitter<IClipboardCopyEvent>());
	public readonly onWillCut: Event<IClipboardCopyEvent> = this._onWillCut.event;

	private readonly _onWillPaste = this._register(new Emitter<IClipboardPasteEvent>());
	public readonly onWillPaste: Event<IClipboardPasteEvent> = this._onWillPaste.event;

	// Dom nodes
	private readonly _linesContent: FastDomNode<HTMLElement>;
	public readonly domNode: FastDomNode<HTMLElement>;
	private readonly _overflowGuardContainer: FastDomNode<HTMLElement>;

	// Actual mutable state
	private _renderAnimationFrame: IDisposable | null;
	private _ownerID: string;

	constructor(
		editorContainer: HTMLElement,
		ownerID: string,
		commandDelegate: ICommandDelegate,
		configuration: IEditorConfiguration,
		colorTheme: IColorTheme,
		model: IViewModel,
		userInputEvents: ViewUserInputEvents,
		overflowWidgetsDomNode: HTMLElement | undefined,
		@IInstantiationService private readonly _instantiationService: IInstantiationService,
		@IUserInteractionService private readonly _userInteractionService: IUserInteractionService,
	) {
		super();
		this._ownerID = ownerID;

		this._widgetFocusTracker = this._register(
			new CodeEditorWidgetFocusTracker(editorContainer, overflowWidgetsDomNode, this._userInteractionService)
		);
		this._register(this._widgetFocusTracker.onChange(() => {
			this._context.viewModel.setHasWidgetFocus(this._widgetFocusTracker.hasFocus());
		}));

		this._selections = [new Selection(1, 1, 1, 1)];
		this._renderAnimationFrame = null;

		this._overflowGuardContainer = createFastDomNode(document.createElement('div'));
		PartFingerprints.write(this._overflowGuardContainer, PartFingerprint.OverflowGuard);
		this._overflowGuardContainer.setClassName('overflow-guard');

		this._viewController = new ViewController(configuration, model, userInputEvents, commandDelegate);

		// The view context is passed on to most classes (basically to reduce param. counts in ctors)
		this._context = new ViewContext(configuration, colorTheme, model);

		// Ensure the view is the first event handler in order to update the layout
		this._context.addEventHandler(this);

		this._viewParts = [];

		// Keyboard handler
		this._editContextEnabled = this._context.configuration.options.get(EditorOption.effectiveEditContext);
		this._accessibilitySupport = this._context.configuration.options.get(EditorOption.accessibilitySupport);
		this._editContext = this._instantiateEditContext();
		this._connectEditContextClipboardEvents();

		this._viewParts.push(this._editContext);

		// These two dom nodes must be constructed up front, since references are needed in the layout provider (scrolling & co.)
		this._linesContent = createFastDomNode(document.createElement('div'));
		this._linesContent.setClassName('lines-content' + ' monaco-editor-background');
		this._linesContent.setPosition('absolute');

		this.domNode = createFastDomNode(document.createElement('div'));
		this.domNode.setClassName(this._getEditorClassName());
		// Set role 'code' for better screen reader support https://github.com/microsoft/vscode/issues/93438
		this.domNode.setAttribute('role', 'code');

		if (this._context.configuration.options.get(EditorOption.experimentalGpuAcceleration) === 'on') {
			this._viewGpuContext = this._instantiationService.createInstance(ViewGpuContext, this._context);
		}

		// View Zones
		this._viewZones = new ViewZones(this._context);
		this._viewParts.push(this._viewZones);

		// Content widgets
		this._contentWidgets = new ViewContentWidgets(this._context, this.domNode);
		this._viewParts.push(this._contentWidgets);

		// Overlay widgets
		this._overlayWidgets = new ViewOverlayWidgets(this._context, this.domNode);
		this._viewParts.push(this._overlayWidgets);

		// -------------- Wire dom nodes up

		this._linesContent.appendChild(this._viewZones.domNode);
		this._linesContent.appendChild(this._contentWidgets.domNode);
		this._overflowGuardContainer.appendChild(this._linesContent);
		this._overflowGuardContainer.appendChild(this._viewZones.marginDomNode);
		if (this._viewGpuContext) {
			this._overflowGuardContainer.appendChild(this._viewGpuContext.canvas);
		}
		this._overflowGuardContainer.appendChild(this._overlayWidgets.getDomNode());
		this.domNode.appendChild(this._overflowGuardContainer);

		if (overflowWidgetsDomNode) {
			overflowWidgetsDomNode.appendChild(this._contentWidgets.overflowingContentWidgetsDomNode.domNode);
			overflowWidgetsDomNode.appendChild(this._overlayWidgets.overflowingOverlayWidgetsDomNode.domNode);
		} else {
			this.domNode.appendChild(this._contentWidgets.overflowingContentWidgetsDomNode);
			this.domNode.appendChild(this._overlayWidgets.overflowingOverlayWidgetsDomNode);
		}

		this._applyLayout();

		// Pointer handler
		this._pointerHandler = this._register(new PointerHandler(this._context, this._viewController, this._createPointerHandlerHelper()));
	}

	private _instantiateEditContext(): AbstractEditContext {
		const usingExperimentalEditContext = this._context.configuration.options.get(EditorOption.effectiveEditContext);
		if (usingExperimentalEditContext) {
			return this._instantiationService.createInstance(NativeEditContext, this._ownerID, this._context, this._overflowGuardContainer, this._viewController, this._createTextAreaHandlerHelper());
		} else {
			return this._instantiationService.createInstance(TextAreaEditContext, this._ownerID, this._context, this._overflowGuardContainer, this._viewController, this._createTextAreaHandlerHelper());
		}
	}

	private _updateEditContext(): void {
		const editContextEnabled = this._context.configuration.options.get(EditorOption.effectiveEditContext);
		const accessibilitySupport = this._context.configuration.options.get(EditorOption.accessibilitySupport);
		if (this._editContextEnabled === editContextEnabled && this._accessibilitySupport === accessibilitySupport) {
			return;
		}
		this._editContextEnabled = editContextEnabled;
		this._accessibilitySupport = accessibilitySupport;
		const isEditContextFocused = this._editContext.isFocused();
		const indexOfEditContext = this._viewParts.indexOf(this._editContext);
		this._editContext.dispose();
		this._editContext = this._instantiateEditContext();
		this._connectEditContextClipboardEvents();
		if (isEditContextFocused) {
			this._editContext.focus();
		}
		if (indexOfEditContext !== -1) {
			this._viewParts.splice(indexOfEditContext, 1, this._editContext);
		}
	}

	private _connectEditContextClipboardEvents(): void {
		// Dispose old listeners
		this._editContextClipboardListeners.clear();

		// Connect to current edit context's clipboard events
		this._editContextClipboardListeners.add(this._editContext.onWillCopy(e => this._onWillCopy.fire(e)));
		this._editContextClipboardListeners.add(this._editContext.onWillCut(e => this._onWillCut.fire(e)));
		this._editContextClipboardListeners.add(this._editContext.onWillPaste(e => this._onWillPaste.fire(e)));
	}

	private _createPointerHandlerHelper(): IPointerHandlerHelper {
		return {
			viewDomNode: this.domNode.domNode,
			linesContentDomNode: this._linesContent.domNode,
			viewLinesDomNode: this._linesContent.domNode,

			focusTextArea: () => {
				this.focus();
			},

			dispatchTextAreaEvent: (event: CustomEvent) => {
				this._editContext.domNode.domNode.dispatchEvent(event);
			},

			getLastRenderData: (): PointerHandlerLastRenderData => {
				// The DOM view-cursors renderer is gone; no cursor render data.
				const lastTextareaPosition = this._editContext.getLastRenderData();
				return new PointerHandlerLastRenderData([], lastTextareaPosition);
			},
			renderNow: (): void => {
				this.render(true, false);
			},
			shouldSuppressMouseDownOnViewZone: (viewZoneId: string) => {
				return this._viewZones.shouldSuppressMouseDownOnViewZone(viewZoneId);
			},
			shouldSuppressMouseDownOnWidget: (widgetId: string) => {
				return this._contentWidgets.shouldSuppressMouseDownOnWidget(widgetId);
			},
			getPositionFromDOMInfo: (spanNode: HTMLElement, offset: number) => {
				// Text rendering is owned by the Go engine; there is no DOM view-line.
				return null;
			},

			visibleRangeForPosition: (lineNumber: number, column: number) => {
				// Text rendering is owned by the Go engine; there is no DOM view-line.
				return null;
			},

			getLineWidth: (lineNumber: number) => {
				// Text rendering is owned by the Go engine; there is no DOM view-line.
				return 0;
			}
		};
	}

	private _createTextAreaHandlerHelper(): IVisibleRangeProvider {
		return {
			visibleRangeForPosition: (position: Position) => {
				// Text rendering is owned by the Go engine; there is no DOM view-line.
				return null;
			},
			linesVisibleRangesForRange: (range: Range, includeNewLines: boolean): LineVisibleRanges[] | null => {
				// Text rendering is owned by the Go engine; there is no DOM view-line.
				return null;
			}
		};
	}

	private _applyLayout(): void {
		const options = this._context.configuration.options;
		const layoutInfo = options.get(EditorOption.layoutInfo);

		this.domNode.setWidth(layoutInfo.width);
		this.domNode.setHeight(layoutInfo.height);

		this._overflowGuardContainer.setWidth(layoutInfo.width);
		this._overflowGuardContainer.setHeight(layoutInfo.height);

		// https://stackoverflow.com/questions/38905916/content-in-google-chrome-larger-than-16777216-px-not-being-rendered
		this._linesContent.setWidth(16777216);
		this._linesContent.setHeight(16777216);
	}

	private _getEditorClassName() {
		const focused = this._editContext.isFocused() ? ' focused' : '';
		return this._context.configuration.options.get(EditorOption.editorClassName) + ' ' + getThemeTypeSelector(this._context.theme.type) + focused;
	}

	// --- begin event handlers
	public override handleEvents(events: viewEvents.ViewEvent[]): void {
		super.handleEvents(events);
		this._scheduleRender();
	}
	public override onConfigurationChanged(e: viewEvents.ViewConfigurationChangedEvent): boolean {
		this.domNode.setClassName(this._getEditorClassName());
		this._updateEditContext();
		this._applyLayout();
		return false;
	}
	public override onCursorStateChanged(e: viewEvents.ViewCursorStateChangedEvent): boolean {
		this._selections = e.selections;
		return false;
	}
	public override onDecorationsChanged(e: viewEvents.ViewDecorationsChangedEvent): boolean {
		return false;
	}
	public override onFocusChanged(e: viewEvents.ViewFocusChangedEvent): boolean {
		this.domNode.setClassName(this._getEditorClassName());
		return false;
	}
	public override onThemeChanged(e: viewEvents.ViewThemeChangedEvent): boolean {
		this._context.theme.update(e.theme);
		this.domNode.setClassName(this._getEditorClassName());
		return false;
	}

	// --- end event handlers

	public override dispose(): void {
		if (this._renderAnimationFrame !== null) {
			this._renderAnimationFrame.dispose();
			this._renderAnimationFrame = null;
		}

		// Dispose clipboard event listeners
		this._editContextClipboardListeners.dispose();

		this._contentWidgets.overflowingContentWidgetsDomNode.domNode.remove();
		this._overlayWidgets.overflowingOverlayWidgetsDomNode.domNode.remove();

		this._context.removeEventHandler(this);
		this._viewGpuContext?.dispose();

		// Destroy view parts
		for (const viewPart of this._viewParts) {
			viewPart.dispose();
		}

		super.dispose();
	}

	private _scheduleRender(): void {
		if (this._store.isDisposed) {
			throw new BugIndicatingError();
		}
		if (this._renderAnimationFrame === null) {
			// TODO: workaround fix for https://github.com/microsoft/vscode/issues/229825
			if (this._editContext instanceof NativeEditContext) {
				this._editContext.setEditContextOnDomNode();
			}
			const rendering = this._createCoordinatedRendering();
			this._renderAnimationFrame = EditorRenderingCoordinator.INSTANCE.scheduleCoordinatedRendering({
				window: dom.getWindow(this.domNode?.domNode),
				prepareRenderText: () => {
					if (this._store.isDisposed) {
						throw new BugIndicatingError();
					}
					try {
						return rendering.prepareRenderText();
					} finally {
						this._renderAnimationFrame = null;
					}
				},
				renderText: (viewportData: ViewportData) => {
					if (this._store.isDisposed) {
						throw new BugIndicatingError();
					}
					return rendering.renderText(viewportData);
				},
				prepareRender: (viewParts: ViewPart[], ctx: RenderingContext) => {
					if (this._store.isDisposed) {
						throw new BugIndicatingError();
					}
					return rendering.prepareRender(viewParts, ctx);
				},
				render: (viewParts: ViewPart[], ctx: RestrictedRenderingContext) => {
					if (this._store.isDisposed) {
						throw new BugIndicatingError();
					}
					return rendering.render(viewParts, ctx);
				}
			});
		}
	}

	private _flushAccumulatedAndRenderNow(): void {
		const rendering = this._createCoordinatedRendering();
		const viewportData = safeInvokeNoArg(() => rendering.prepareRenderText());
		if (!viewportData) {
			return;
		}
		const data = safeInvokeNoArg(() => rendering.renderText(viewportData));
		if (!data) {
			return;
		}
		const [viewParts, ctx] = data;
		safeInvokeNoArg(() => rendering.prepareRender(viewParts, ctx));
		safeInvokeNoArg(() => rendering.render(viewParts, ctx));
	}

	private _getViewPartsToRender(): ViewPart[] {
		const result: ViewPart[] = [];
		let resultLen = 0;
		for (const viewPart of this._viewParts) {
			if (viewPart.shouldRender()) {
				result[resultLen++] = viewPart;
			}
		}
		return result;
	}

	private _createCoordinatedRendering() {
		return {
			prepareRenderText: () => {
				inputLatency.onRenderStart();

				if (!this.domNode.domNode.isConnected) {
					return null;
				}

				const viewPartsToRender = this._getViewPartsToRender();
				if (viewPartsToRender.length === 0) {
					// Nothing to render
					return null;
				}

				const partialViewportData = this._context.viewLayout.getLinesViewportData();
				this._context.viewModel.setViewport(partialViewportData.startLineNumber, partialViewportData.endLineNumber, partialViewportData.centeredLineNumber);

				const viewportData = new ViewportData(
					this._selections,
					partialViewportData,
					this._context.viewLayout.getWhitespaceViewportData(),
					this._context.viewModel
				);

				for (const viewPart of this._viewParts) {
					if (viewPart.shouldRender()) {
						viewPart.onBeforeRender(viewportData);
					}
				}

				return viewportData;
			},
			renderText: (viewportData: ViewportData): [ViewPart[], RenderingContext] => {
				const viewPartsToRender = this._getViewPartsToRender();

				return [viewPartsToRender, new RenderingContext(this._context.viewLayout, viewportData, new NullViewLines())];
			},
			prepareRender: (viewPartsToRender: ViewPart[], ctx: RenderingContext) => {
				for (const viewPart of viewPartsToRender) {
					viewPart.prepareRender(ctx);
				}
			},
			render: (viewPartsToRender: ViewPart[], ctx: RestrictedRenderingContext) => {
				for (const viewPart of viewPartsToRender) {
					viewPart.render(ctx);
					viewPart.onDidRender();
				}
			}
		};
	}

	// --- BEGIN CodeEditor helpers

	public delegateVerticalScrollbarPointerDown(browserEvent: PointerEvent): void {
		// The scrollbar is rendered by the Go engine; nothing to do here.
	}

	public delegateScrollFromMouseWheelEvent(browserEvent: IMouseWheelEvent) {
		// The scrollbar is rendered by the Go engine; nothing to do here.
	}

	public restoreState(scrollPosition: { scrollLeft: number; scrollTop: number }): void {
		this._context.viewModel.viewLayout.setScrollPosition({
			scrollTop: scrollPosition.scrollTop,
			scrollLeft: scrollPosition.scrollLeft
		}, ScrollType.Immediate);
		this._context.viewModel.visibleLinesStabilized();
	}

	public getOffsetForColumn(modelLineNumber: number, modelColumn: number): number {
		// Text rendering is owned by the Go engine; there is no DOM view-line to measure.
		return -1;
	}

	public getLineWidth(modelLineNumber: number): number {
		// Text rendering is owned by the Go engine; there is no DOM view-line to measure.
		return 0;
	}

	public resetLineWidthCaches(): void {
		// Text rendering is owned by the Go engine; nothing to reset.
	}

	public getTargetAtClientPoint(clientX: number, clientY: number): IMouseTarget | null {
		const mouseTarget = this._pointerHandler.getTargetAtClientPoint(clientX, clientY);
		if (!mouseTarget) {
			return null;
		}
		return ViewUserInputEvents.convertViewToModelMouseTarget(mouseTarget, this._context.viewModel.coordinatesConverter);
	}

	public createOverviewRuler(cssClassName: string): OverviewRuler {
		return new OverviewRuler(this._context, cssClassName);
	}

	public change(callback: (changeAccessor: IViewZoneChangeAccessor) => unknown): void {
		this._viewZones.changeViewZones(callback);
		this._scheduleRender();
	}

	public render(now: boolean, everything: boolean): void {
		if (everything) {
			// Force everything to render...
			for (const viewPart of this._viewParts) {
				viewPart.forceShouldRender();
			}
		}
		if (now) {
			this._flushAccumulatedAndRenderNow();
		} else {
			this._scheduleRender();
		}
	}

	public writeScreenReaderContent(reason: string): void {
		this._editContext.writeScreenReaderContent(reason);
	}

	public focus(): void {
		this._editContext.focus();
	}

	public isFocused(): boolean {
		return this._editContext.isFocused();
	}

	public isWidgetFocused(): boolean {
		return this._widgetFocusTracker.hasFocus();
	}

	public refreshFocusState() {
		this._editContext.refreshFocusState();
		this._widgetFocusTracker.refreshState();
	}

	public setAriaOptions(options: IEditorAriaOptions): void {
		this._editContext.setAriaOptions(options);
	}

	public addContentWidget(widgetData: IContentWidgetData): void {
		this._contentWidgets.addWidget(widgetData.widget);
		this.layoutContentWidget(widgetData);
		this._scheduleRender();
	}

	public layoutContentWidget(widgetData: IContentWidgetData): void {
		this._contentWidgets.setWidgetPosition(
			widgetData.widget,
			widgetData.position?.position ?? null,
			widgetData.position?.secondaryPosition ?? null,
			widgetData.position?.preference ?? null,
			widgetData.position?.positionAffinity ?? null
		);
		if (this._contentWidgets.shouldRender()) {
			this._scheduleRender();
		}
	}

	public removeContentWidget(widgetData: IContentWidgetData): void {
		this._contentWidgets.removeWidget(widgetData.widget);
		this._scheduleRender();
	}

	public addOverlayWidget(widgetData: IOverlayWidgetData): void {
		this._overlayWidgets.addWidget(widgetData.widget);
		this.layoutOverlayWidget(widgetData);
		this._scheduleRender();
	}

	public layoutOverlayWidget(widgetData: IOverlayWidgetData): void {
		const shouldRender = this._overlayWidgets.setWidgetPosition(widgetData.widget, widgetData.position);
		if (shouldRender) {
			this._scheduleRender();
		}
	}

	public removeOverlayWidget(widgetData: IOverlayWidgetData): void {
		this._overlayWidgets.removeWidget(widgetData.widget);
		this._scheduleRender();
	}

	public addGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// The glyph margin is rendered by the Go engine; nothing to do here.
	}

	public layoutGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// The glyph margin is rendered by the Go engine; nothing to do here.
	}

	public removeGlyphMarginWidget(widgetData: IGlyphMarginWidgetData): void {
		// The glyph margin is rendered by the Go engine; nothing to do here.
	}

	// --- END CodeEditor helpers

}

function safeInvokeNoArg<T>(func: () => T): T | null {
	try {
		return func();
	} catch (e) {
		onUnexpectedError(e);
		return null;
	}
}

interface ICoordinatedRendering {
	readonly window: CodeWindow;
	prepareRenderText(): ViewportData | null;
	renderText(viewportData: ViewportData): [ViewPart[], RenderingContext];
	prepareRender(viewParts: ViewPart[], ctx: RenderingContext): void;
	render(viewParts: ViewPart[], ctx: RestrictedRenderingContext): void;
}

class EditorRenderingCoordinator {

	public static INSTANCE = new EditorRenderingCoordinator();

	private _coordinatedRenderings: ICoordinatedRendering[] = [];
	private _animationFrameRunners = new Map<CodeWindow, IDisposable>();

	private constructor() { }

	scheduleCoordinatedRendering(rendering: ICoordinatedRendering): IDisposable {
		this._coordinatedRenderings.push(rendering);
		this._scheduleRender(rendering.window);
		return {
			dispose: () => {
				const renderingIndex = this._coordinatedRenderings.indexOf(rendering);
				if (renderingIndex === -1) {
					return;
				}
				this._coordinatedRenderings.splice(renderingIndex, 1);

				if (this._coordinatedRenderings.length === 0) {
					// There are no more renderings to coordinate => cancel animation frames
					for (const [_, disposable] of this._animationFrameRunners) {
						disposable.dispose();
					}
					this._animationFrameRunners.clear();
				}
			}
		};
	}

	private _scheduleRender(window: CodeWindow): void {
		if (!this._animationFrameRunners.has(window)) {
			const runner = () => {
				this._animationFrameRunners.delete(window);
				this._onRenderScheduled();
			};
			this._animationFrameRunners.set(window, dom.runAtThisOrScheduleAtNextAnimationFrame(window, runner, 100));
		}
	}

	private _onRenderScheduled(): void {
		const coordinatedRenderings = this._coordinatedRenderings.slice(0);
		this._coordinatedRenderings = [];

		const viewportDatas: (ViewportData | null)[] = [];
		for (let i = 0, len = coordinatedRenderings.length; i < len; i++) {
			const rendering = coordinatedRenderings[i];
			viewportDatas[i] = safeInvokeNoArg(() => rendering.prepareRenderText());
		}

		const datas: ([ViewPart[], RenderingContext] | null)[] = [];
		for (let i = 0, len = coordinatedRenderings.length; i < len; i++) {
			const rendering = coordinatedRenderings[i];
			const viewportData = viewportDatas[i];
			if (!viewportData) {
				datas[i] = null;
				continue;
			}
			datas[i] = safeInvokeNoArg(() => rendering.renderText(viewportData));
		}

		for (let i = 0, len = coordinatedRenderings.length; i < len; i++) {
			const rendering = coordinatedRenderings[i];
			const data = datas[i];
			if (!data) {
				continue;
			}
			const [viewParts, ctx] = data;
			safeInvokeNoArg(() => rendering.prepareRender(viewParts, ctx));
		}

		for (let i = 0, len = coordinatedRenderings.length; i < len; i++) {
			const rendering = coordinatedRenderings[i];
			const data = datas[i];
			if (!data) {
				continue;
			}
			const [viewParts, ctx] = data;
			safeInvokeNoArg(() => rendering.render(viewParts, ctx));
		}
	}
}

class CodeEditorWidgetFocusTracker extends Disposable {

	private _hasDomElementFocus: boolean;
	private readonly _domFocusTracker: dom.IFocusTracker;
	private readonly _overflowWidgetsDomNode: dom.IFocusTracker | undefined;

	private readonly _onChange: Emitter<void> = this._register(new Emitter<void>());
	public readonly onChange: Event<void> = this._onChange.event;

	private _overflowWidgetsDomNodeHasFocus: boolean;

	private _hadFocus: boolean | undefined = undefined;

	constructor(domElement: HTMLElement, overflowWidgetsDomNode: HTMLElement | undefined, userInteractionService: IUserInteractionService) {
		super();

		this._hasDomElementFocus = false;
		this._domFocusTracker = this._register(userInteractionService.createDomFocusTracker(domElement));

		this._overflowWidgetsDomNodeHasFocus = false;

		this._register(this._domFocusTracker.onDidFocus(() => {
			this._hasDomElementFocus = true;
			this._update();
		}));
		this._register(this._domFocusTracker.onDidBlur(() => {
			this._hasDomElementFocus = false;
			this._update();
		}));

		if (overflowWidgetsDomNode) {
			this._overflowWidgetsDomNode = this._register(userInteractionService.createDomFocusTracker(overflowWidgetsDomNode));
			this._register(this._overflowWidgetsDomNode.onDidFocus(() => {
				this._overflowWidgetsDomNodeHasFocus = true;
				this._update();
			}));
			this._register(this._overflowWidgetsDomNode.onDidBlur(() => {
				this._overflowWidgetsDomNodeHasFocus = false;
				this._update();
			}));
		}
	}

	private _update() {
		const focused = this._hasDomElementFocus || this._overflowWidgetsDomNodeHasFocus;
		if (this._hadFocus !== focused) {
			this._hadFocus = focused;
			this._onChange.fire(undefined);
		}
	}

	public hasFocus(): boolean {
		return this._hadFocus ?? false;
	}

	public refreshState(): void {
		this._domFocusTracker.refreshState();
		this._overflowWidgetsDomNode?.refreshState?.();
	}
}
