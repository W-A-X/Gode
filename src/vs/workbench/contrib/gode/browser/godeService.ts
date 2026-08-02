/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../../../platform/instantiation/common/instantiation.js';
import { Emitter, Event } from '../../../../base/common/event.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { ILogService } from '../../../../platform/log/common/log.js';
import { ITextModel } from '../../../../editor/common/model.js';
import { IPos, IRange } from '../common/godeProtocol.js';
import { GodeRenderer } from './godeRenderer.js';

export const IGodeService = createDecorator<IGodeService>('godeService');

export interface IGodeService {
	readonly _serviceBrand: undefined;

	// Engine management
	initialize(): Promise<void>;
	shutdown(): void;
	isEngineReady(): boolean;

	// Document operations
	openDocument(uri: string, text: string): Promise<void>;
	updateDocument(uri: string, text: string): void;
	closeDocument(uri: string): void;
	getDocumentContent(uri: string): Promise<string>;

	// Selection operations
	setSelection(uri: string, anchor: IPos, active: IPos): void;
	getSelection(uri: string): { anchor: IPos; active: IPos } | null;

	// Events
	readonly onDidChangeSelection: Event<{ uri: string; anchor: IPos; active: IPos }>;
	readonly onDidChangeContent: Event<{ uri: string; range: IRange; text: string }>;
	readonly onEngineReady: Event<void>;
	readonly onEngineError: Event<Error>;

	// Backend registration
	registerTextModel(uri: string, model: ITextModel): void;
	unregisterTextModel(uri: string): void;
}

export class GodeService extends Disposable implements IGodeService {

	readonly _serviceBrand: undefined;

	private renderer: GodeRenderer | null = null;
	private engineReady = false;

	private readonly documents = new Map<string, string>(); // uri -> content
	private readonly selections = new Map<string, { anchor: IPos; active: IPos }>();
	private readonly textModels = new Map<string, ITextModel>();

	private readonly _onDidChangeSelection = this._register(new Emitter<{ uri: string; anchor: IPos; active: IPos }>());
	readonly onDidChangeSelection = this._onDidChangeSelection.event;

	private readonly _onDidChangeContent = this._register(new Emitter<{ uri: string; range: IRange; text: string }>());
	readonly onDidChangeContent = this._onDidChangeContent.event;

	private readonly _onEngineReady = this._register(new Emitter<void>());
	readonly onEngineReady = this._onEngineReady.event;

	private readonly _onEngineError = this._register(new Emitter<Error>());
	readonly onEngineError = this._onEngineError.event;

	constructor(
		@ILogService private readonly logService: ILogService
	) {
		super();
	}

	async initialize(): Promise<void> {
		if (this.engineReady) {
			return;
		}

		this.renderer = new GodeRenderer(this.logService);

		this._register(this.renderer.onReady(() => {
			this.engineReady = true;
			this._onEngineReady.fire();
			this.logService.info('[gode] Engine initialized successfully');
		}));

		this._register(this.renderer.onError((err) => {
			this._onEngineError.fire(err);
			this.logService.error(`[gode] Engine error: ${err.message}`);
		}));

		this._register(this.renderer.onSelectionChanged((sel) => {
			// Find the active document (simplified - single document support)
			const activeUri = this.getActiveUri();
			if (activeUri) {
				this.selections.set(activeUri, sel);
				this._onDidChangeSelection.fire({ uri: activeUri, ...sel });
			}
		}));

		this._register(this.renderer.onEdited((edit) => {
			const activeUri = this.getActiveUri();
			if (activeUri) {
				this._onDidChangeContent.fire({ uri: activeUri, range: edit.range, text: edit.editText });
				// Update the cached document content
				const current = this.documents.get(activeUri) || '';
				const updated = this.applyEdit(current, edit.range, edit.editText);
				this.documents.set(activeUri, updated);

				// Sync back to VS Code model if registered
				const model = this.textModels.get(activeUri);
				if (model) {
					// Note: full sync would require more complex logic
					this.logService.debug(`[gode] Edit applied to ${activeUri}: ${edit.editText}`);
				}
			}
		}));

		await this.renderer.connect();
	}

	shutdown(): void {
		if (this.renderer) {
			this.renderer.disconnect();
			this.renderer = null;
			this.engineReady = false;
		}
		this.documents.clear();
		this.selections.clear();
		this.textModels.clear();
	}

	isEngineReady(): boolean {
		return this.engineReady;
	}

	async openDocument(uri: string, text: string): Promise<void> {
		this.documents.set(uri, text);
		this.selections.set(uri, { anchor: { line: 1, column: 1 }, active: { line: 1, column: 1 } });

		if (this.renderer && this.engineReady) {
			this.renderer.setText(text);
		}
	}

	updateDocument(uri: string, text: string): void {
		this.documents.set(uri, text);

		if (this.renderer && this.engineReady) {
			this.renderer.setText(text);
		}
	}

	closeDocument(uri: string): void {
		this.documents.delete(uri);
		this.selections.delete(uri);
		this.textModels.delete(uri);
	}

	async getDocumentContent(uri: string): Promise<string> {
		if (this.renderer && this.engineReady) {
			return await this.renderer.getContent(Date.now());
		}
		return this.documents.get(uri) || '';
	}

	setSelection(uri: string, anchor: IPos, active: IPos): void {
		this.selections.set(uri, { anchor, active });

		if (this.renderer && this.engineReady) {
			this.renderer.setSelection(anchor, active);
		}
	}

	getSelection(uri: string): { anchor: IPos; active: IPos } | null {
		return this.selections.get(uri) || null;
	}

	registerTextModel(uri: string, model: ITextModel): void {
		this.textModels.set(uri, model);
	}

	unregisterTextModel(uri: string): void {
		this.textModels.delete(uri);
	}

	private getActiveUri(): string | null {
		// Simplified: return the first document URI
		for (const uri of this.documents.keys()) {
			return uri;
		}
		return null;
	}

	private applyEdit(text: string, range: IRange, editText: string): string {
		const lines = text.split('\n');
		const startLine = range.start.line - 1;
		const startCol = range.start.column - 1;
		const endLine = range.end.line - 1;
		const endCol = range.end.column - 1;

		// Build new content
		const newLines: string[] = [];

		// Lines before start
		for (let i = 0; i < startLine; i++) {
			newLines.push(lines[i]);
		}

		// Start line - before selection
		if (startLine < lines.length) {
			newLines.push(lines[startLine].substring(0, startCol) + editText);
		}

		// Lines between start and end
		for (let i = startLine + 1; i < endLine; i++) {
			newLines.push(''); // These lines are replaced by editText
		}

		// End line - after selection
		if (endLine < lines.length) {
			const endLineContent = lines[endLine];
			const afterSelection = endLineContent.substring(endCol);
			// Check if we need to append to the last inserted line
			if (newLines.length > 0) {
				newLines[newLines.length - 1] = newLines[newLines.length - 1] + afterSelection;
			} else {
				newLines.push(afterSelection);
			}
		}

		// Lines after end
		for (let i = endLine + 1; i < lines.length; i++) {
			newLines.push(lines[i]);
		}

		return newLines.join('\n');
	}
}
