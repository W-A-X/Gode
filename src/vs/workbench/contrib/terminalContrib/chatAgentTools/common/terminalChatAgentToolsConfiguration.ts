/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export const enum TerminalChatAgentToolsSettingId {
	EnableAutoApprove = 'chat.tools.terminal.enableAutoApprove',
	AutoApprove = 'chat.tools.terminal.autoApprove',
	AutoApproveWorkspaceNpmScripts = 'chat.tools.terminal.autoApproveWorkspaceNpmScripts',
	IgnoreDefaultAutoApproveRules = 'chat.tools.terminal.ignoreDefaultAutoApproveRules',
	BlockDetectedFileWrites = 'chat.tools.terminal.blockDetectedFileWrites',
	ShellIntegrationTimeout = 'chat.tools.terminal.shellIntegrationTimeout',
	OutputLocation = 'chat.tools.terminal.outputLocation',
	AgentSandboxLinuxFileSystem = 'chat.agent.sandbox.fileSystem.linux',
	AgentSandboxMacFileSystem = 'chat.agent.sandbox.fileSystem.mac',
	AgentSandboxWindowsFileSystem = 'chat.agent.sandbox.fileSystem.windows',
	AgentSandboxAdvancedRuntime = 'chat.agent.sandbox.advanced.runtime',
	PreventShellHistory = 'chat.tools.terminal.preventShellHistory',
	EnforceTimeoutFromModel = 'chat.tools.terminal.enforceTimeoutFromModel',
	IdleSilenceTimeoutMs = 'chat.tools.terminal.idleSilenceTimeoutMs',
	DetachBackgroundProcesses = 'chat.tools.terminal.detachBackgroundProcesses',
	BackgroundNotifications = 'chat.tools.terminal.backgroundNotifications',
	OutputDeltas = 'chat.tools.terminal.outputDeltas',
	OutputCompaction = 'chat.tools.terminal.outputCompaction',
	IdlePollInterval = 'chat.tools.terminal.idlePollInterval',
	TerminalProfileLinux = 'chat.tools.terminal.terminalProfile.linux',
	TerminalProfileMacOs = 'chat.tools.terminal.terminalProfile.osx',
	TerminalProfileWindows = 'chat.tools.terminal.terminalProfile.windows',
	DeprecatedAutoApproveCompatible = 'chat.agent.terminal.autoApprove',
	DeprecatedAutoApprove1 = 'chat.agent.terminal.allowList',
	DeprecatedAutoApprove2 = 'chat.agent.terminal.denyList',
	DeprecatedAutoApprove3 = 'github.copilot.chat.agent.terminal.allowList',
	DeprecatedAutoApprove4 = 'github.copilot.chat.agent.terminal.denyList',
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const terminalChatAgentToolsConfiguration: any = {};
