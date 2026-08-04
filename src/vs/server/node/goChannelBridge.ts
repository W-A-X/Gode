/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ILogService } from '../../platform/log/common/log.js';
import { GoBackendClient } from './goBackendClient.js';
import { GoFileSystemChannel } from './goFileSystemChannel.js';
import { GoTerminalChannel } from './goTerminalChannel.js';
import { IServerChannel } from '../../base/parts/ipc/common/ipc.js';
import { RemoteAgentConnectionContext } from '../../platform/remote/common/remoteAgentEnvironment.js';
import { REMOTE_FILE_SYSTEM_CHANNEL_NAME } from '../../workbench/services/remote/common/remoteFileSystemProviderClient.js';
import { REMOTE_TERMINAL_CHANNEL_NAME } from '../../workbench/contrib/terminal/common/remote/remoteTerminalChannel.js';
import { Event } from '../../base/common/event.js';

export class GoChannelBridge {

	private goClient: GoBackendClient;

	constructor(private readonly logService: ILogService) {
		this.goClient = new GoBackendClient(logService);
	}

	isAvailable(): boolean {
		return this.goClient.isConnected();
	}

	getChannel(channelName: string): IServerChannel<RemoteAgentConnectionContext> | null {
		if (!this.goClient.isConnected()) {
			return null;
		}

		switch (channelName) {
			case REMOTE_FILE_SYSTEM_CHANNEL_NAME:
				return new GoFileSystemChannel(this.goClient, this.logService);
			case REMOTE_TERMINAL_CHANNEL_NAME:
				return new GoTerminalChannel(this.goClient, this.logService);
			default:
				return null;
		}
	}

	getConnectionEvent(): Event<boolean> {
		return this.goClient.onDidChangeConnection;
	}
}