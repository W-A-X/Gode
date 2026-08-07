/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../../../platform/instantiation/common/instantiation.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { Event, Emitter } from '../../../../base/common/event.js';

export interface IAgentHostTerminalService {
	readonly _serviceBrand: undefined;
	readonly onDidOpenAgentTerminal: Event<string>;
	readonly onDidFocusTerminal: Event<string>;
	openTerminal(): Promise<void>;
	focusTerminal(): Promise<void>;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	onDidChangeTerminals: Event<any>;
}

export const IAgentHostTerminalService = createDecorator<IAgentHostTerminalService>('agentHostTerminalService');

export class AgentHostTerminalService extends Disposable implements IAgentHostTerminalService {
	declare readonly _serviceBrand: undefined;

	private readonly _onDidOpenAgentTerminal = this._register(new Emitter<string>());
	readonly onDidOpenAgentTerminal = this._onDidOpenAgentTerminal.event;

	private readonly _onDidFocusTerminal = this._register(new Emitter<string>());
	readonly onDidFocusTerminal = this._onDidFocusTerminal.event;

	private readonly _onDidChangeTerminals = this._register(new Emitter<void>());
	readonly onDidChangeTerminals = this._onDidChangeTerminals.event;

	async openTerminal(): Promise<void> {
		// stub
	}

	async focusTerminal(): Promise<void> {
		// stub
	}
}
