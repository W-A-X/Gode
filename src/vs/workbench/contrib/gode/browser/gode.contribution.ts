/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { registerWorkbenchContribution2, WorkbenchPhase } from '../../../common/contributions.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configuration.js';
import { Extensions as ConfigExtensions, IConfigurationRegistry } from '../../../../platform/configuration/common/configurationRegistry.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { Registry } from '../../../../platform/registry/common/platform.js';
import { setGodeViewFactory } from '../../../../editor/browser/widget/codeEditor/codeEditorWidget.js';
import { GodeView } from './godeView.js';

Registry.as<IConfigurationRegistry>(ConfigExtensions.Configuration).registerConfiguration({
	id: 'gode',
	order: 100,
	type: 'object',
	properties: {
		'gode.enabled': {
			type: 'boolean',
			default: true,
			description: 'Render the editor with the Go (gogpu/ui) offscreen engine instead of the default DOM view. Requires the gode-engine process (GODE_ENGINE_PATH).'
		}
	}
});

class GodeContribution {

	constructor(
		@IInstantiationService _instantiationService: IInstantiationService,
		@IConfigurationService configurationService: IConfigurationService,
	) {
		if (!configurationService.getValue<boolean>('gode.enabled')) {
			return;
		}

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
}

registerWorkbenchContribution2(
	'workbench.contrib.gode',
	GodeContribution,
	WorkbenchPhase.BlockRestore
);
