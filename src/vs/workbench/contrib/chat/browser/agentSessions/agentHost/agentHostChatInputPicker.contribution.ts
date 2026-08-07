/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export class OpenAgentHostFolderPickerAction {
	static readonly ID = 'workbench.action.chat.openAgentHostFolderPicker';
}

export class OpenAgentHostModePickerAction {
	static readonly ID = 'workbench.action.chat.openAgentHostModePicker';
}

export class OpenAgentHostAutoApprovePickerAction {
	static readonly ID = 'workbench.action.chat.openAgentHostAutoApprovePicker';
}

export class OpenAgentHostPermissionModePickerAction {
	static readonly ID = 'workbench.action.chat.openAgentHostPermissionModePicker';
}

export class OpenAgentHostCodexApprovalsPickerAction {
	static readonly ID = 'workbench.action.chat.openAgentHostCodexApprovalsPicker';
}

export function getAgentHostPickerProperty(actionId: string): string | undefined {
	return actionId;
}
