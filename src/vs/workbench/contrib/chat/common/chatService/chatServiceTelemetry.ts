/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export interface ChatRequestTelemetry {
	// no-op telemetry removed
}

export class ChatServiceTelemetry {
	constructor() { }
	notifyUserAction(_action: unknown): void { }
	retrievedFollowups(_agentId: string | undefined, _command: string, _count: number | undefined): void { }
}
