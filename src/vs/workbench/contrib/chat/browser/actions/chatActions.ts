/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { localize } from '../../../../../nls.js';
import { MenuId } from '../../../../../platform/actions/common/actions.js';
import { ServicesAccessor } from '../../../../../platform/instantiation/common/instantiation.js';

export const CHAT_CATEGORY = localize2('chat.category', 'Chat');
export const ACTION_ID_NEW_CHAT = 'workbench.action.chat.newChat';
export const CHAT_OPEN_ACTION_ID = 'workbench.action.chat.open';
export const CHAT_SETUP_ACTION_ID = 'workbench.action.chat.triggerSetup';
export const CHAT_SETUP_SUPPORT_ANONYMOUS_ACTION_ID = 'workbench.action.chat.triggerSetupSupportAnonymousAction';
export const GENERATE_AGENT_INSTRUCTIONS_COMMAND_ID = 'workbench.action.chat.generateAgentInstructions';
export const GENERATE_PROMPT_COMMAND_ID = 'workbench.action.chat.generatePrompt';
export const GENERATE_SKILL_COMMAND_ID = 'workbench.action.chat.generateSkill';
export const GENERATE_AGENT_COMMAND_ID = 'workbench.action.chat.generateAgent';
export const INSERT_FORK_CONVERSATION_COMMAND_ID = 'workbench.action.chat.insertForkConversationCommand';
export const INSERT_TROUBLESHOOT_COMMAND_ID = 'workbench.action.chat.insertTroubleshootCommand';
export const CHAT_CONFIG_MENU_ID = new MenuId('workbench.chat.menu.config');

export interface IClearEditingSessionConfirmationOptions {
	openNewSession: boolean;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function getOpenChatActionIdForMode(mode: any): string {
	return `workbench.action.chat.open${mode?.name?.get?.() ?? 'Ask'}`;
}

export async function handleCurrentEditingSession(model: unknown, phrase: string | undefined, dialogService: unknown): Promise<boolean> {
	return true;
}

export async function handleModeSwitch(accessor: ServicesAccessor, currentModeKind: unknown, targetModeKind: unknown, requestCount?: number, model?: unknown): Promise<boolean> {
	return true;
}

export async function clearChatSessionPreservingType(accessor: ServicesAccessor, widget: unknown, sessionType: string | undefined): Promise<void> {
	// stub
}
