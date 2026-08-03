/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { registerWorkbenchContribution2, WorkbenchPhase } from '../../../common/contributions.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configurationService.js';
import { Extensions as ConfigExtensions, IConfigurationRegistry } from '../../../../platform/configuration/common/configurationRegistry.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { Registry } from '../../../../platform/registry/common/platform.js';
import { setGodeViewFactory } from '../../../../editor/browser/widget/codeEditor/codeEditorWidget.js';
import { GodeView } from './godeView.js';
import { GodeServicesManager } from './godeServicesManager.js';

// Register Gode configuration
Registry.as<IConfigurationRegistry>(ConfigExtensions.Configuration).registerConfiguration({
        id: 'gode',
        order: 100,
        type: 'object',
        properties: {
                'gode.enabled': {
                        type: 'boolean',
                        default: true,
                        description: 'Render the editor with the Go (gogpu/ui) offscreen engine instead of the default DOM view. Requires the gode-engine process (GODE_ENGINE_PATH).'
                },
                'gode.services.enabled': {
                        type: 'boolean',
                        default: true,
                        description: 'Use Go-based backend services for file operations and Git integration instead of native Node.js/Electron implementations.'
                },
                'gode.services.filePort': {
                        type: 'number',
                        default: 47811,
                        description: 'Port number for the Go file service (file-service).'
                },
                'gode.services.gitPort': {
                        type: 'number',
                        default: 47812,
                        description: 'Port number for the Go Git service (git-service).'
                }
        }
});

/**
 * Main Gode contribution that registers:
 * 1. The Go-based editor view (GodeView)
 * 2. The Go-based services manager (file + git)
 */
class GodeContribution {

        private _servicesManager: GodeServicesManager | null = null;

        constructor(
                @IInstantiationService private readonly instantiationService: IInstantiationService,
                @IConfigurationService configurationService: IConfigurationService,
        ) {
                // Initialize editor rendering (GodeView)
                if (configurationService.getValue<boolean>('gode.enabled')) {
                        setGodeViewFactory((args) => {
                                return new GodeView(
                                        args.editorContainer,
                                        args.ownerID,
                                        args.commandDelegate,
                                        args.configuration,
                                        args.colorTheme,
                                        args.model,
                                        args.userInputEvents,
                                        args.overflowWidgetsDomNode,
                                        args.instantiationService,
                                        args.userInteractionService
                                );
                        });
                }

                // Initialize Go-based services (file + git)
                if (configurationService.getValue<boolean>('gode.services.enabled')) {
                        this._servicesManager = this.instantiationService.createInstance(GodeServicesManager);
                }
        }
}

registerWorkbenchContribution2(
        'workbench.contrib.gode',
        GodeContribution,
        WorkbenchPhase.BlockRestore
);

// Export service accessors for other modules
export function getGodeServices(): GodeServicesManager | null {
        // This will be set by the contribution constructor
        // Access via dependency injection when possible
        return null;
}
