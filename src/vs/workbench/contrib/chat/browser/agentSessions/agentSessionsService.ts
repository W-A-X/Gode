/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { URI } from '../../../../../base/common/uri.js';
import { Emitter, Event } from '../../../../../base/common/event.js';
import { Disposable } from '../../../../../base/common/lifecycle.js';
import { createDecorator, InstantiationType, registerSingleton } from '../../../../../platform/instantiation/common/instantiation.js';

export interface IAgentSession {
	readonly _serviceBrand: undefined;
	readonly resource: URI;
	readonly sessionKey: string;
	readonly metadata: unknown;
	isRead: boolean;
	canBeRead(): boolean;
	setRead(value: boolean): void;
}

export interface IAgentSessionsModel {
	readonly _serviceBrand: undefined;
	readonly onDidChangeSessions: Event<unknown>;
	readonly sessions: IAgentSession[];
}

export interface IAgentSessionsService {
	readonly _serviceBrand: undefined;
	readonly model: IAgentSessionsModel;
	readonly onDidChangeSessionArchivedState: Event<IAgentSession>;
	getSession(resource: URI): IAgentSession | undefined;
}

export const IAgentSessionsService = createDecorator<IAgentSessionsService>('agentSessions');

class AgentSessionsModel implements IAgentSessionsModel {
	declare readonly _serviceBrand: undefined;
	readonly onDidChangeSessions: Event<unknown> = new Emitter<unknown>().event;
	readonly sessions: IAgentSession[] = [];
}

export class AgentSessionsService extends Disposable implements IAgentSessionsService {
	declare readonly _serviceBrand: undefined;

	private readonly _onDidChangeSessionArchivedState = this._register(new Emitter<IAgentSession>());
	readonly onDidChangeSessionArchivedState = this._onDidChangeSessionArchivedState.event;

	readonly model: IAgentSessionsModel = new AgentSessionsModel();

	getSession(_resource: URI): IAgentSession | undefined {
		return undefined;
	}
}

registerSingleton(IAgentSessionsService, AgentSessionsService, InstantiationType.Delayed);
