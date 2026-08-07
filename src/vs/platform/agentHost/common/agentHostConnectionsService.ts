/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../instantiation/common/instantiation.js';

export const LOCAL_AGENT_HOST_SCHEME_PREFIX = 'agent-host-';

export const AMBIENT_AGENT_HOST_AUTHORITY = 'local';

export interface IAgentHostConnectionInfo {
	readonly providerId: string;
	readonly authority: string;
	readonly label: string;
	readonly description: string;
}

export interface IAgentHostSessionResolution {
	readonly sessionType: string;
	readonly sessionKey: string;
}

export const IAgentHostConnectionsService = createDecorator<IAgentHostConnectionsService>('agentHostConnectionsService');

export interface IAgentHostConnectionsService {
	readonly _serviceBrand: undefined;
	readonly connections: IAgentHostConnectionInfo[];
	readonly connectionsContainer: HTMLElement | undefined;
}
