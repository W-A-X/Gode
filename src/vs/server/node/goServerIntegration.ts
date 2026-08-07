/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { IServerChannel } from '../../base/parts/ipc/common/ipc.js';
import { ILogService } from '../../platform/log/common/log.js';
import { GoBackendClient } from './goBackendClient.js';
import { GoFileSystemChannel } from './goFileSystemChannel.js';
import { GoTerminalChannel } from './goTerminalChannel.js';
import { RemoteAgentConnectionContext } from '../../platform/remote/common/remoteAgentEnvironment.js';
import { REMOTE_FILE_SYSTEM_CHANNEL_NAME } from '../../workbench/services/remote/common/remoteFileSystemProviderClient.js';
import { REMOTE_TERMINAL_CHANNEL_NAME } from '../../workbench/contrib/terminal/common/remote/remoteTerminalChannel.js';

export interface GoServerIntegrationOptions {
	enabled: boolean;
	goBackendPort?: number;
	goBackendHost?: string;
}

export class GoServerIntegration {

	private readonly client: GoBackendClient;
	private readonly enabled: boolean;

	constructor(
		private readonly logService: ILogService,
		options: GoServerIntegrationOptions,
	) {
		this.enabled = options.enabled;
		this.client = new GoBackendClient(logService);

		if (this.enabled) {
			this.logService.info('[GoServerIntegration] Go backend integration enabled');
		} else {
			this.logService.info('[GoServerIntegration] Go backend integration disabled, using TypeScript backend');
		}
	}

	isGoBackendAvailable(): boolean {
		return this.enabled && this.client.isConnected();
	}

	getChannel(channelName: string, fallbackChannel: IServerChannel<RemoteAgentConnectionContext>): IServerChannel<RemoteAgentConnectionContext> {
		if (!this.isGoBackendAvailable()) {
			return fallbackChannel;
		}

		switch (channelName) {
			case REMOTE_FILE_SYSTEM_CHANNEL_NAME:
				this.logService.info('[GoServerIntegration] Delegating file system to Go backend');
				return new GoFileSystemChannel(this.client, this.logService) as IServerChannel<RemoteAgentConnectionContext>;
			case REMOTE_TERMINAL_CHANNEL_NAME:
				this.logService.info('[GoServerIntegration] Delegating terminal to Go backend');
				return new GoTerminalChannel(this.client) as IServerChannel<RemoteAgentConnectionContext>;
			default:
				return fallbackChannel;
		}
	}

	getClient(): GoBackendClient {
		return this.client;
	}
}