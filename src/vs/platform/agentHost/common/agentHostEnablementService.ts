/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../instantiation/common/instantiation.js';
import { RawContextKey } from '../../contextkey/common/contextkey.js';
import { type IObservable } from '../../../../base/common/observable.js';
import { localize } from '../../../nls.js';

export const AGENT_HOST_ENABLED_CONTEXT_KEY = new RawContextKey<boolean>('agentHostEnabled', false, { type: 'boolean', description: localize('agentHostEnabled', "Whether the local agent host process is enabled.") });

export const IAgentHostEnablementService = createDecorator<IAgentHostEnablementService>('agentHostEnablementService');

export interface IAgentHostEnablementService {
	readonly _serviceBrand: undefined;
	/**
	 * Whether Agent Host features are enabled in this runtime.
	 * This can transition from `false` to `true` when a startup experiment is applied or AI features are explicitly enabled.
	 */
	readonly enabled: IObservable<boolean>;
}
