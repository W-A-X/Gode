/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { Disposable, DisposableStore, toDisposable } from '../../../../base/common/lifecycle.js';
import { mainWindow } from '../../../../base/browser/window.js';
import { addDisposableListener } from '../../../../base/browser/dom.js';
import { GodeEngineClient } from './godeEngineClient.js';
import { GODE_ENGINE_PORT, IActivityBarItem, ISidebarItem, IPanelTab, IAuxiliaryTab, IStatusItem, ITitleState } from '../common/godeProtocol.js';
import { IWorkbenchLayoutService, Parts } from '../../../services/layout/browser/layoutService.js';
import { IEditorService } from '../../../services/editor/common/editorService.js';
import { IViewDescriptorService, ViewContainerLocation } from '../../../common/views.js';
import { IStatusbarService } from '../../../services/statusbar/browser/statusbar.js';
import { ITitleService } from '../../../services/title/browser/titleService.js';
import { ICommandService } from '../../../../platform/commands/common/commands.js';
import { INotificationService } from '../../../../platform/notification/common/notification.js';
import { URI } from '../../../../base/common/uri.js';
import { IWorkspaceContextService } from '../../../../platform/workspace/common/workspace.js';

/**
 * GodeWorkbenchView renders the entire IDE workbench using gogpu/ui via the
 * gode-engine process. It replaces all DOM-based parts (titlebar, activitybar,
 * sidebar, panel, auxiliarybar, statusbar) with a single full-window canvas.
 *
 * The extension host continues to run independently via RPC; this view only
 * replaces the rendering layer, not the extension host communication.
 */
export class GodeWorkbenchView extends Disposable {

	private readonly _canvas: HTMLCanvasElement;
	private readonly _ctx: CanvasRenderingContext2D;
	private readonly _client: GodeEngineClient;
	private readonly _disposables: DisposableStore;
	private _focused: boolean = false;
	private _lastWidth: number = 0;
	private _lastHeight: number = 0;

	constructor(
		private readonly _container: HTMLElement,
		@IWorkbenchLayoutService private readonly layoutService: IWorkbenchLayoutService,
		@IEditorService private readonly editorService: IEditorService,
		@IViewDescriptorService private readonly viewDescriptorService: IViewDescriptorService,
		@IStatusbarService private readonly statusbarService: IStatusbarService,
		@ITitleService private readonly titleService: ITitleService,
		@ICommandService private readonly commandService: ICommandService,
		@INotificationService private readonly notificationService: INotificationService,
		@IWorkspaceContextService private readonly workspaceContextService: IWorkspaceContextService,
	) {
		super();
		this._disposables = this._register(new DisposableStore());

		// Create full-window canvas
		this._canvas = document.createElement('canvas');
		this._canvas.id = 'gode-workbench-canvas';
		this._canvas.style.position = 'absolute';
		this._canvas.style.inset = '0';
		this._canvas.style.width = '100%';
		this._canvas.style.height = '100%';
		this._canvas.style.display = 'block';
		this._canvas.setAttribute('tabindex', '0');
		this._container.appendChild(this._canvas);

		const ctx = this._canvas.getContext('2d');
		if (!ctx) {
			throw new Error('GodeWorkbenchView: cannot create 2d canvas context');
		}
		this._ctx = ctx;

		// Connect to the gode-engine
		this._client = new GodeEngineClient(GODE_ENGINE_PORT, this._canvas, this._ctx);

		this._setupEngineCallbacks();
		this._setupInputForwarding();
		this._setupVSCodeStateSync();
		this._setupLayoutObserver();
	}

	private _setupEngineCallbacks(): void {
		// Activity bar selection -> open the corresponding view
		this._client.onActivityBarSelected((id) => {
			this._handleActivityBarSelection(id);
		});

		// Sidebar file selection -> open the file in the editor
		this._client.onSidebarItemSelected((path) => {
			this._handleSidebarFileSelection(path);
		});

		// Sidebar directory toggle -> expand/collapse
		this._client.onSidebarItemToggle((path, expanded) => {
			this._handleSidebarToggle(path, expanded);
		});

		// Panel tab selection -> show the corresponding panel
		this._client.onPanelTabSelected((id) => {
			this._handlePanelTabSelection(id);
		});

		// Auxiliary tab selection -> show the corresponding auxiliary panel
		this._client.onAuxiliaryTabSelected((id) => {
			this._handleAuxiliaryTabSelection(id);
		});

		// Title bar actions -> toggle parts
		this._client.onTitlebarAction((action) => {
			this._handleTitlebarAction(action);
		});

		// Command palette request -> show VS Code quick input
		this._client.onCommandPaletteRequested(() => {
			this._handleCommandPaletteRequest();
		});

		// Command selected from palette -> execute
		this._client.onCommandSelected((id) => {
			this.commandService.executeCommand(id);
		});

		// Status item clicked -> handle status bar action
		this._client.onStatusItemClicked((id) => {
			this._handleStatusItemClick(id);
		});

		// Input bar submit -> handle command input
		this._client.onInputBarSubmit((text) => {
			this._handleInputBarSubmit(text);
		});

		// Engine ready -> sync initial state
		this._client.onEngineReady(() => {
			this._syncFullState();
		});

		// Notification from engine
		this._client.onNotification((message, level) => {
			if (level === 'error') {
				this.notificationService.error(message);
			} else if (level === 'warning') {
				this.notificationService.warn(message);
			} else {
				this.notificationService.info(message);
			}
		});
	}

	private _setupInputForwarding(): void {
		// Forward all mouse events to the engine
		this._disposables.add(addDisposableListener(this._canvas, 'mousedown', (e: MouseEvent) => {
			this._client.sendMouse('press', e);
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'mouseup', (e: MouseEvent) => {
			this._client.sendMouse('release', e);
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'mousemove', (e: MouseEvent) => {
			this._client.sendMouse((e.buttons & 1) ? 'drag' : 'move', e);
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'wheel', (e: WheelEvent) => {
			e.preventDefault();
			const dpr = mainWindow.devicePixelRatio || 1;
			this._client.sendWheelDelta(e.deltaX * dpr, e.deltaY * dpr);
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'dblclick', (e: MouseEvent) => {
			this._client.sendMouse('double_click', e);
		}));

		// Forward keyboard events to the engine
		this._disposables.add(addDisposableListener(this._canvas, 'keydown', (e: KeyboardEvent) => {
			// Let global shortcuts (Ctrl/Cmd+P, Ctrl/Cmd+Shift+P, etc.) pass through
			if (this._isGlobalShortcut(e)) {
				return;
			}
			e.preventDefault();
			e.stopImmediatePropagation();
			this._client.sendKey('press', e);
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'keyup', (e: KeyboardEvent) => {
			if (this._isGlobalShortcut(e)) {
				return;
			}
			this._client.sendKey('release', e);
		}));

		// Focus tracking
		this._disposables.add(addDisposableListener(this._canvas, 'focus', () => {
			this._focused = true;
			this._client.focus();
		}));

		this._disposables.add(addDisposableListener(this._canvas, 'blur', () => {
			this._focused = false;
		}));
	}

	private _setupVSCodeStateSync(): void {
		// Sync activity bar items when views change
		this._disposables.add(this.viewDescriptorService.onDidChangeViewContainers(() => {
			this._syncActivityBar();
		}));

		// Sync title when window title changes
		this._disposables.add(this.titleService.windowTitle.onDidChange(() => {
			this._syncTitle();
		}));

		// Sync status bar items
		this._disposables.add(this.statusbarService.onDidChangeEntryVisibility(() => {
			this._syncStatusBar();
		}));

		// Sync layout state when parts visibility changes
		this._disposables.add(this.layoutService.onDidChangePartVisibility(() => {
			this._syncLayoutState();
		}));

		// Sync panel tabs when terminal/output panels change
		this._disposables.add(this.editorService.onDidActiveEditorChange(() => {
			this._syncPanelTabs();
		}));
	}

	private _setupLayoutObserver(): void {
		// Observe resize and update the engine viewport
		const resizeObserver = new ResizeObserver(() => {
			this._updateViewport();
		});
		resizeObserver.observe(this._container);
		this._disposables.add(toDisposable(() => resizeObserver.disconnect()));

		// Initial viewport setup
		this._updateViewport();
	}

	private _updateViewport(): void {
		const rect = this._container.getBoundingClientRect();
		if (rect.width <= 0 || rect.height <= 0) {
			return;
		}
		const dpr = mainWindow.devicePixelRatio || 1;
		const w = Math.ceil(rect.width * dpr);
		const h = Math.ceil(rect.height * dpr);
		if (w === this._lastWidth && h === this._lastHeight) {
			return;
		}
		this._lastWidth = w;
		this._lastHeight = h;
		this._client.setWorkbenchViewport(w, h, dpr);
	}

	// --- State synchronization ---

	private _syncFullState(): void {
		this._syncActivityBar();
		this._syncSidebarItems();
		this._syncPanelTabs();
		this._syncAuxiliaryTabs();
		this._syncStatusBar();
		this._syncTitle();
		this._syncLayoutState();
	}

	private _syncActivityBar(): void {
		const containers = this.viewDescriptorService.getViewContainersByLocation(ViewContainerLocation.Sidebar);
		const items: IActivityBarItem[] = containers.map(c => ({
			id: c.id,
			name: typeof c.title === 'string' ? c.title : c.title.value,
			icon: c.icon ? c.icon.toString() : 'file',
			badge_count: 0,
			is_visible: this.viewDescriptorService.getViewContainerModel(c).activeViewDescriptors.length > 0,
		}));
		this._client.setActivityBarItems(items);
	}

	private _syncSidebarItems(): void {
		const workspace = this.workspaceContextService.getWorkspace();
		if (workspace.folders.length === 0) {
			this._client.setSidebarItems([]);
			return;
		}
		// Build tree items from workspace folders
		const items: ISidebarItem[] = workspace.folders.map(folder => ({
			id: folder.uri.toString(),
			name: folder.name,
			path: folder.uri.path,
			is_directory: true,
			is_expanded: true,
			children: [],
			icon: 'folder',
		}));
		this._client.setSidebarItems(items);
	}

	private _syncPanelTabs(): void {
		const tabs: IPanelTab[] = [
			{ id: 'terminal', name: 'Terminal', icon: 'terminal', is_active: false, content_type: 'terminal' },
			{ id: 'output', name: 'Output', icon: 'output', is_active: false, content_type: 'output' },
			{ id: 'problems', name: 'Problems', icon: 'warning', is_active: false, content_type: 'problems' },
			{ id: 'debug', name: 'Debug Console', icon: 'debug-console', is_active: false, content_type: 'debug' },
		];
		this._client.setPanelTabs(tabs);
	}

	private _syncAuxiliaryTabs(): void {
		const tabs: IAuxiliaryTab[] = [
			{ id: 'chat', name: 'Chat', icon: 'comment-discussion', is_active: false },
			{ id: 'agents', name: 'Agents', icon: 'robot', is_active: false },
		];
		this._client.setAuxiliaryTabs(tabs);
	}

	private _syncStatusBar(): void {
		const items: IStatusItem[] = [];
		// Add branch info
		items.push({
			id: 'branch',
			text: 'main',
			icon: 'git-branch',
			alignment: 'left',
			tooltip: 'Current branch',
		});
		// Add cursor position from active text editor
		const activeTextEditorControl = this.editorService.activeTextEditorControl;
		if (activeTextEditorControl && 'getSelection' in activeTextEditorControl) {
			const selection = activeTextEditorControl.getSelection();
			if (selection) {
				items.push({
					id: 'cursor',
					text: `Ln ${selection.selectionStartLineNumber}, Col ${selection.selectionStartColumn}`,
					alignment: 'right',
				});
			}
		}
		// Add language mode
		items.push({
			id: 'language',
			text: this.editorService.activeEditor?.getName() || 'Plain Text',
			alignment: 'right',
		});
		this._client.setStatusItems(items);
	}

	private _syncTitle(): void {
		const state: ITitleState = {
			title: this.titleService.windowTitle.value,
			subtitle: '',
			sidebar_visible: this.layoutService.isVisible(Parts.SIDEBAR_PART),
			panel_visible: this.layoutService.isVisible(Parts.PANEL_PART),
			auxiliary_visible: this.layoutService.isVisible(Parts.AUXILIARYBAR_PART),
		};
		this._client.setTitleState(state);
	}

	private _syncLayoutState(): void {
		this._client.setLayoutState(
			this.layoutService.isVisible(Parts.ACTIVITYBAR_PART),
			this.layoutService.isVisible(Parts.SIDEBAR_PART),
			this.layoutService.isVisible(Parts.PANEL_PART),
			this.layoutService.isVisible(Parts.AUXILIARYBAR_PART),
			this.layoutService.isVisible('workbench.parts.statusbar' as any),
		);
	}

	// --- Event handlers ---

	private async _handleActivityBarSelection(id: string): Promise<void> {
		// Toggle the view container visibility
		const container = this.viewDescriptorService.getViewContainerById(id);
		if (container) {
			const location = this.viewDescriptorService.getViewContainerLocation(container);
			if (location === ViewContainerLocation.Sidebar) {
				await this.commandService.executeCommand('workbench.action.toggleSidebarVisibility');
			}
		}
	}

	private async _handleSidebarFileSelection(path: string): Promise<void> {
		try {
			const uri = URI.file(path);
			await this.editorService.openEditor({ resource: uri });
		} catch (err) {
			this.notificationService.error(`Failed to open file: ${path}`);
		}
	}

	private _handleSidebarToggle(path: string, expanded: boolean): void {
		// The engine handles the visual toggle; we just acknowledge it
		// In a full implementation, this would update the file explorer model
	}

	private async _handlePanelTabSelection(id: string): Promise<void> {
		switch (id) {
			case 'terminal':
				await this.commandService.executeCommand('workbench.action.terminal.toggleTerminal');
				break;
			case 'output':
				await this.commandService.executeCommand('workbench.output.action.switchToOutput');
				break;
			case 'problems':
				await this.commandService.executeCommand('workbench.panel.markers.view.focus');
				break;
			case 'debug':
				await this.commandService.executeCommand('workbench.debug.action.toggleRepl');
				break;
		}
	}

	private async _handleAuxiliaryTabSelection(id: string): Promise<void> {
		switch (id) {
			case 'chat':
				await this.commandService.executeCommand('workbench.action.chat.open');
				break;
			case 'agents':
				await this.commandService.executeCommand('workbench.action.openAgentsView');
				break;
		}
	}

	private async _handleTitlebarAction(action: string): Promise<void> {
		switch (action) {
			case 'toggle_sidebar':
				await this.commandService.executeCommand('workbench.action.toggleSidebarVisibility');
				break;
			case 'toggle_panel':
				await this.commandService.executeCommand('workbench.action.togglePanel');
				break;
			case 'toggle_auxiliary':
				await this.commandService.executeCommand('workbench.action.toggleAuxiliaryBar');
				break;
			case 'toggle_command_center':
				await this.commandService.executeCommand('workbench.action.toggleCommandCenter');
				break;
		}
	}

	private async _handleCommandPaletteRequest(): Promise<void> {
		await this.commandService.executeCommand('workbench.action.quickOpen');
	}

	private _handleStatusItemClick(id: string): void {
		switch (id) {
			case 'branch':
				this.commandService.executeCommand('git.checkout');
				break;
			case 'cursor':
				this.commandService.executeCommand('workbench.action.gotoLine');
				break;
			case 'language':
				this.commandService.executeCommand('workbench.action.editor.changeLanguageMode');
				break;
		}
	}

	private _handleInputBarSubmit(text: string): void {
		// Execute the submitted command
		this.commandService.executeCommand(text);
	}

	// --- Helpers ---

	private _isGlobalShortcut(e: KeyboardEvent): boolean {
		// Let Ctrl/Cmd shortcuts pass through to VS Code's global handler
		if (e.ctrlKey || e.metaKey) {
			return true;
		}
		// Let F1 (command palette) pass through
		if (e.key === 'F1') {
			return true;
		}
		return false;
	}

	// --- Public API ---

	public focus(): void {
		this._canvas.focus();
		this._client.focus();
	}

	public get isFocused(): boolean {
		return this._focused;
	}

	public override dispose(): void {
		this._client.dispose();
		this._disposables.dispose();
		super.dispose();
	}
}


