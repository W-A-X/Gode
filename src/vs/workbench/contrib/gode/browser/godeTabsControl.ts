/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { $, Dimension, clearNode } from '../../../../base/browser/dom.js';
import { Emitter, Event } from '../../../../base/common/event.js';
import { Disposable, IDisposable, toDisposable } from '../../../../base/common/lifecycle.js';
import { IEditorGroupsView, IEditorGroupView, IEditorGroupTitleHeight } from '../editor.js';
import { IEditorInput, EditorInput } from '../../../common/editor/editorInput.js';
import { IEditorTabsControl } from './editorTabsControl.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IEditorPartOptions, IEditorPartsView } from '../../../common/editor.js';
import { IReadonlyEditorGroupModel } from '../../../common/editor/editorGroupModel.js';
import { GodeEngineClient } from './godeEngineClient.js';
import { ITabInfo, IGodeEvent } from '../common/godeProtocol.js';

/**
 * GodeTabsControl replaces the standard DOM-based editor tabs with a
 * Canvas-based implementation rendered by the Go (gogpu/ui) engine.
 *
 * It communicates tab state to the Go engine via WebSocket and receives
 * rendered RGBA frames that are drawn onto a canvas element.
 */
export class GodeTabsControl extends Disposable implements IEditorTabsControl {
        private readonly _element: HTMLElement;
        private readonly _canvas: HTMLCanvasElement;
        private _dimension: Dimension | undefined;
        private _client: GodeEngineClient;
        private _currentTabs: ITabInfo[] = [];
        private _activeTabIdx: number = -1;

        private readonly _onDidTabSelected = this._register(new Emitter<number>());
        readonly onDidTabSelected: Event<number> = this._onDidTabSelected.event;

        private readonly _onDidTabClose = this._register(new Emitter<number>());
        readonly onDidTabClose: Event<number> = this._onDidTabClose.event;

        constructor(
                parent: HTMLElement,
                private readonly editorPartsView: IEditorPartsView,
                private readonly groupsView: IEditorGroupsView,
                private readonly groupView: IEditorGroupView,
                private readonly model: IReadonlyEditorGroupModel,
                @IInstantiationService instantiationService: IInstantiationService,
        ) {
                super();

                this._element = parent;
                this._client = new GodeEngineClient();
                this._canvas = document.createElement('canvas');
                this._canvas.style.width = '100%';
                this._canvas.style.height = '100%';
                this._canvas.style.display = 'block';

                // Clear existing content and add canvas
                clearNode(parent);
                parent.appendChild(this._canvas);

                // Listen for tab events from Go engine
                this._register(this._client.onEvent((event: IGodeEvent) => {
                        this.handleEngineEvent(event);
                }));

                // Initialize connection
                this._client.connect().then(() => {
                        // Send initial viewport once connected
                        if (this._dimension) {
                                this.sendTabViewport();
                        }
                        // Sync current tabs
                        this.syncTabs();
                });
        }

        private handleEngineEvent(event: IGodeEvent): void {
                switch (event.evt) {
                        case 'tab_frame':
                                this.renderTabFrame(event);
                                break;
                        case 'tab_selected':
                                if (event.tab_selected_idx !== undefined) {
                                        this._onDidTabSelected.fire(event.tab_selected_idx);
                                }
                                break;
                        case 'tab_close':
                                if (event.tab_close_idx !== undefined) {
                                        this._onDidTabClose.fire(event.tab_close_idx);
                                }
                                break;
                }
        }

        private renderTabFrame(event: IGodeEvent): void {
                if (!event.tab_data || !event.tab_width || !event.tab_height) {
                        return;
                }

                const width = event.tab_width;
                const height = event.tab_height;

                // Set canvas size
                if (this._canvas.width !== width || this._canvas.height !== height) {
                        this._canvas.width = width;
                        this._canvas.height = height;
                }

                const ctx = this._canvas.getContext('2d');
                if (!ctx) return;

                // Convert base64 or Uint8Array to ImageData
                let data: Uint8Array;
                if (typeof event.tab_data === 'string') {
                        const binaryString = atob(event.tab_data);
                        data = new Uint8Array(binaryString.length);
                        for (let i = 0; i < binaryString.length; i++) {
                                data[i] = binaryString.charCodeAt(i);
                        }
                } else {
                        data = event.tab_data;
                }

                const imageData = new ImageData(new Uint8ClampedArray(data), width, height);
                ctx.putImageData(imageData, 0, 0);
        }

        private sendTabViewport(): void {
                if (!this._dimension) return;

                const dpr = window.devicePixelRatio || 1;
                this._client.sendCommand({
                        cmd: 'tab_viewport',
                        tab_viewport: {
                                width: Math.round(this._dimension.width * dpr),
                                height: Math.round(this._dimension.height * dpr),
                                scale: dpr
                        }
                });
        }

        private syncTabs(): void {
                this._client.sendCommand({
                        cmd: 'set_tabs',
                        tabs: {
                                tabs: this._currentTabs,
                                active_idx: this._activeTabIdx
                        }
                });
        }

        private buildTabInfo(editor: IEditorInput, isActive: boolean): ITabInfo {
                const isDirty = editor.isDirty() && !editor.isSaving();
                const label = editor.getName();
                const description = editor.getDescription() || '';

                // Determine icon name based on resource
                let iconName = '';
                const resource = editor.resource;
                if (resource) {
                        const ext = resource.path.split('.').pop()?.toLowerCase();
                        iconName = `file-type-${ext || 'text'}`;
                }

                return {
                        label,
                        description,
                        icon_name: iconName,
                        is_dirty: isDirty,
                        is_active: isActive,
                        is_pinned: false // TODO: support pinned tabs
                };
        }

        // --- IEditorTabsControl implementation ---

        openEditor(editor: IEditorInput, options?: { active?: boolean; pinned?: boolean; index?: number }): boolean {
                // Rebuild all tabs from model
                const editors = this.model.editors;
                this._currentTabs = editors.map((e, i) => this.buildTabInfo(e, e === editor));
                this._activeTabIdx = editors.indexOf(editor);

                this.syncTabs();
                return true; // Tab list changed
        }

        openEditors(editors: IEditorInput[]): boolean {
                const allEditors = this.model.editors;
                this._currentTabs = allEditors.map((e, i) => {
                        const isActive = this.model.activeEditor === e;
                        return this.buildTabInfo(e, isActive);
                });
                this._activeTabIdx = allEditors.indexOf(this.model.activeEditor);

                this.syncTabs();
                return true;
        }

        beforeCloseEditor(editor: IEditorInput): void {
                // Nothing to do before close
        }

        closeEditor(editor: IEditorInput): void {
                const idx = this._currentTabs.findIndex(t => t.label === editor.getName());
                if (idx >= 0) {
                        this._currentTabs.splice(idx, 1);
                        if (this._activeTabIdx >= this._currentTabs.length) {
                                this._activeTabIdx = this._currentTabs.length - 1;
                        }
                        this.syncTabs();
                }
        }

        closeEditors(editors: IEditorInput[]): void {
                const labelsToRemove = new Set(editors.map(e => e.getName()));
                this._currentTabs = this._currentTabs.filter(t => !labelsToRemove.has(t.label));
                this._activeTabIdx = Math.min(this._activeTabIdx, this._currentTabs.length - 1);
                this.syncTabs();
        }

        moveEditor(editor: IEditorInput, fromIndex: number, targetIndex: number, stickyStateChange: boolean): void {
                // Rebuild tabs after reorder
                this.openEditors([]);
        }

        pinEditor(editor: IEditorInput): void {
                // Update pinned state
                const idx = this._currentTabs.findIndex(t => t.label === editor.getName());
                if (idx >= 0) {
                        this._currentTabs[idx] = { ...this._currentTabs[idx], is_pinned: true };
                        this.syncTabs();
                }
        }

        stickEditor(editor: IEditorInput): void {
                this.pinEditor(editor);
        }

        unstickEditor(editor: IEditorInput): void {
                const idx = this._currentTabs.findIndex(t => t.label === editor.getName());
                if (idx >= 0) {
                        this._currentTabs[idx] = { ...this._currentTabs[idx], is_pinned: false };
                        this.syncTabs();
                }
        }

        setActive(isActive: boolean): void {
                // Visual update only - handled by CSS/canvas rendering
        }

        updateEditorSelections(): void {
                // Selection doesn't affect tabs visually
        }

        updateEditorLabel(editor: IEditorInput): void {
                const idx = this._currentTabs.findIndex(t => t.label === editor.getName());
                if (idx >= 0) {
                        this._currentTabs[idx] = this.buildTabInfo(editor, this._activeTabIdx === idx);
                        this.syncTabs();
                }
        }

        updateEditorDirty(editor: IEditorInput): void {
                this.updateEditorLabel(editor); // Dirty state is part of tab info
        }

        updateOptions(oldOptions: IEditorPartOptions, newOptions: IEditorPartOptions): void {
                // Options changed - may need to re-render
                this.syncTabs();
        }

        layout(dimensions: { container: Dimension; available: Dimension }): Dimension {
                this._dimension = dimensions.available;

                // Set canvas display size (CSS pixels)
                this._canvas.style.width = `${dimensions.available.width}px`;
                this._canvas.style.height = `${dimensions.available.height}px`;

                // Send updated viewport to engine
                this.sendTabViewport();
                this.syncTabs();

                // Return fixed height for tab bar (35px like VS Code default)
                return new Dimension(dimensions.available.width, 35);
        }

        getHeight(): number {
                return 35; // Standard VS Code tab bar height
        }

        /**
         * Forward a mouse event to the Go engine's tab bar handler.
         * Y coordinate should be negative to indicate tab area.
         */
        forwardMouseEvent(mouseEvent: MouseEvent): void {
                if (!this._dimension) return;

                const rect = this._canvas.getBoundingClientRect();
                const dpr = window.devicePixelRatio || 1;
                const x = (mouseEvent.clientX - rect.left) * dpr;
                const y = (mouseEvent.clientY - rect.top) * dpr;

                // Map mouse event type
                let mouseType: IInputMouse['mouse_type'] = 'move';
                if (mouseEvent.type === 'mousedown') mouseType = 'press';
                else if (mouseEvent.type === 'mouseup') mouseType = 'release';
                else if (mouseEvent.type === 'mousemove' && mouseEvent.buttons > 0) mouseType = 'drag';
                else if (mouseEvent.type === 'dblclick') mouseType = 'double_click';

                let button: 'left' | 'right' | 'middle' = 'left';
                if (mouseEvent.button === 2) button = 'right';
                else if (mouseEvent.button === 1) button = 'middle';

                this._client.sendCommand({
                        cmd: 'input',
                        type: 'mouse',
                        mouse: {
                                mouse_type: mouseType,
                                button,
                                x,
                                y,
                                shift: mouseEvent.shiftKey,
                                ctrl: mouseEvent.ctrlKey,
                                alt: mouseEvent.altKey,
                                super: mouseEvent.metaKey
                        }
                });
        }

        dispose(): void {
                super.dispose();
        }
}

// Import types needed for mouse forwarding
interface IInputMouse {
        readonly mouse_type: 'press' | 'release' | 'move' | 'drag' | 'double_click';
        readonly button: 'left' | 'right' | 'middle';
        readonly x: number;
        readonly y: number;
        readonly shift?: boolean;
        readonly ctrl?: boolean;
        readonly alt?: boolean;
        readonly super?: boolean;
}
