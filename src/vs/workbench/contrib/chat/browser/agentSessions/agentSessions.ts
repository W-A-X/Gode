/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { URI } from '../../../../../base/common/uri.js';
import { localize } from '../../../../../nls.js';
import { Codicon } from '../../../../../base/common/codicons.js';
import { ThemeIcon } from '../../../../../base/common/themables.js';
import { SessionType } from '../../common/chatSessionsService.js';
import { getChatSessionType } from '../../common/model/chatUri.js';

export enum AgentSessionProviders {
	Local = SessionType.Local,
	Background = SessionType.CopilotCLI,
	Cloud = SessionType.CopilotCloud,
	Claude = SessionType.ClaudeCode,
	Codex = SessionType.Codex,
	Growth = SessionType.Growth,
	AgentHostCopilot = SessionType.AgentHostCopilot,
	AgentHostClaude = SessionType.AgentHostClaude,
	AgentHostCodex = SessionType.AgentHostCodex,
}

export type AgentSessionTarget = AgentSessionProviders | (string & {});

export function isBuiltInAgentSessionProvider(provider: AgentSessionTarget): boolean {
	return provider === AgentSessionProviders.Local ||
		provider === AgentSessionProviders.Background ||
		provider === AgentSessionProviders.Cloud ||
		provider === AgentSessionProviders.Claude;
}

export function getAgentSessionProvider(sessionResource: URI | string): AgentSessionProviders | undefined {
	const type = URI.isUri(sessionResource) ? getChatSessionType(sessionResource) : sessionResource;
	switch (type) {
		case AgentSessionProviders.Local:
		case AgentSessionProviders.Background:
		case AgentSessionProviders.Cloud:
		case AgentSessionProviders.Claude:
		case AgentSessionProviders.Codex:
		case AgentSessionProviders.AgentHostCopilot:
		case AgentSessionProviders.AgentHostClaude:
		case AgentSessionProviders.AgentHostCodex:
			return type;
		default:
			return undefined;
	}
}

export function getAgentSessionProviderName(provider: AgentSessionTarget): string {
	switch (provider) {
		case AgentSessionProviders.Local:
			return localize('chat.session.providerLabel.local', "Local");
		case AgentSessionProviders.Background:
			return localize('chat.session.providerLabel.background', "Copilot CLI");
		case AgentSessionProviders.Cloud:
			return localize('chat.session.providerLabel.cloud', "Cloud");
		case AgentSessionProviders.Claude:
		case AgentSessionProviders.AgentHostClaude:
			return 'Claude';
		case AgentSessionProviders.Codex:
		case AgentSessionProviders.AgentHostCodex:
			return 'Codex';
		case AgentSessionProviders.Growth:
			return 'Growth';
		case AgentSessionProviders.AgentHostCopilot:
			return localize('chat.session.providerLabel.agentHostCopilot', "Copilot");
		default:
			return provider;
	}
}

export function getAgentSessionProviderIcon(provider: AgentSessionTarget): ThemeIcon {
	switch (provider) {
		case AgentSessionProviders.Local:
			return Codicon.vm;
		case AgentSessionProviders.Background:
			return Codicon.copilot;
		case AgentSessionProviders.Cloud:
			return Codicon.cloud;
		case AgentSessionProviders.Codex:
		case AgentSessionProviders.AgentHostCodex:
			return Codicon.openai;
		case AgentSessionProviders.Claude:
		case AgentSessionProviders.AgentHostClaude:
			return Codicon.claude;
		case AgentSessionProviders.Growth:
			return Codicon.lightbulb;
		default:
			return Codicon.vm;
	}
}

export function isFirstPartyAgentSessionProvider(provider: AgentSessionTarget): boolean {
	return provider === AgentSessionProviders.Local ||
		provider === AgentSessionProviders.Background ||
		provider === AgentSessionProviders.Cloud ||
		provider === AgentSessionProviders.Claude ||
		provider === AgentSessionProviders.Codex ||
		provider === AgentSessionProviders.Growth;
}

export function getAgentCanContinueIn(provider: AgentSessionTarget, _model: unknown): boolean {
	return isBuiltInAgentSessionProvider(provider);
}

export const CHAT_DELEGATE_TO_AGENT_HOST_SESSION_COMMAND_ID = 'workbench.action.chat.delegateToAgentHostSession';
