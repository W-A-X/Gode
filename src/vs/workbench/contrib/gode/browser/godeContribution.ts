/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { registerSingleton, InstantiationType } from '../../../../platform/instantiation/common/extensions.js';
import { ILogService } from '../../../../platform/log/common/log.js';
import { IEditorGroupsService } from '../../../services/editor/common/editorGroupsService.js';
import { IWorkbenchContribution } from '../../../common/contributions.js';
import { GodeEditorWidget } from './godeEditorWidget.js';
import { GodeService, IGodeService } from './godeService.js';
import { startGodeEngine } from '../electron-main/godeProcessMain.js';

// Register the gode service
registerSingleton(IGodeService, GodeService, InstantiationType.Delayed);

/**
 * GodeContribution integrates the Go-based editor into the VS Code workbench.
 * It manages the lifecycle of the gode engine and editor widgets.
 */
export class GodeContribution implements IWorkbenchContribution {

	static readonly ID = 'gode.contrib.godeContribution';

	private readonly godeService: IGodeService;

	constructor(
		@ILogService private readonly logService: ILogService,
		@IEditorGroupsService private readonly editorGroupsService: IEditorGroupsService
	) {
		this.godeService = new GodeService(logService);

		// Initialize the engine
		this.initializeEngine();

		// Register event handlers
		this.registerEventHandlers();
	}

	private async initializeEngine(): Promise<void> {
		try {
			// Try to start the engine process
			const started = startGodeEngine(this.logService);
			if (!started) {
				this.logService.warn('[gode] Engine not started - GODE_ENGINE_PATH may not be set');
			}

			// Initialize the service (connects to the engine)
			await this.godeService.initialize();
			this.logService.info('[gode] Gode contribution initialized');
		} catch (err) {
			this.logService.error(`[gode] Failed to initialize: ${err}`);
		}
	}

	private registerEventHandlers(): void {
		// Handle selection changes from the engine
		this.godeService.onDidChangeSelection((e) => {
			this.logService.debug(`[gode] Selection changed in ${e.uri}: ${e.anchor.line}:${e.anchor.column} - ${e.active.line}:${e.active.column}`);
		});

		// Handle content changes from the engine
		this.godeService.onDidChangeContent((e) => {
			this.logService.debug(`[gode] Content changed in ${e.uri}`);
			// Note: Full sync with VS Code text models would be implemented here
		});

		// Handle engine ready
		this.godeService.onEngineReady(() => {
			this.logService.info('[gode] Engine is ready');
		});

		// Handle engine errors
		this.godeService.onEngineError((err) => {
			this.logService.error(`[gode] Engine error: ${err.message}`);
		});

		// Handle editor group changes
		this.editorGroupsService.onDidChangeGroupIndex((e) => {
			this.logService.debug(`[gode] Editor group ${e.index} changed`);
		});
	}

	/**
	 * Create a GodeEditorWidget for the given container.
	 */
	createEditorWidget(container: HTMLElement): GodeEditorWidget {
		return new GodeEditorWidget(container, this.logService);
	}

	/**
	 * Open a document in the gode editor.
	 */
	async openDocument(uri: string, text: string): Promise<void> {
		await this.godeService.openDocument(uri, text);
	}

	/**
	 * Close a document.
	 */
	closeDocument(uri: string): void {
		this.godeService.closeDocument(uri);
	}

	dispose(): void {
		this.godeService.shutdown();
	}
}

// Register the contribution
registerGodeContribution();

function registerGodeContribution(): void {

	// This is a simplified registration - the actual registration would go
	// through the VS Code contribution system
	console.log('[gode] GodeContribution registration helper loaded');
}
