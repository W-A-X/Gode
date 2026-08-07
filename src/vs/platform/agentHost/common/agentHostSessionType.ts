/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

const REMOTE_AGENT_HOST_SESSION_TYPE_PREFIX = 'remote-';

export function remoteAgentHostSessionTypeId(connectionAuthority: string, agentProvider: string): string {
	return `${remoteAgentHostSessionTypeAuthorityPrefix(connectionAuthority)}${agentProvider}`;
}

export function remoteAgentHostSessionTypeAuthorityPrefix(connectionAuthority: string): string {
	return `${REMOTE_AGENT_HOST_SESSION_TYPE_PREFIX}${connectionAuthority}-`;
}

export function isRemoteAgentHostSessionType(sessionType: string): boolean {
	return sessionType.startsWith(REMOTE_AGENT_HOST_SESSION_TYPE_PREFIX);
}

export function findRemoteAgentHostSessionTypeAuthority(sessionType: string, connectionAuthorities: Iterable<string>): string | undefined {
	for (const authority of connectionAuthorities) {
		if (sessionType.startsWith(remoteAgentHostSessionTypeAuthorityPrefix(authority))) {
			return authority;
		}
	}
	return undefined;
}

export function parseRemoteAgentHostHarness(sessionType: string): string | undefined {
	const index = sessionType.indexOf('-', REMOTE_AGENT_HOST_SESSION_TYPE_PREFIX.length);
	return index === -1 ? undefined : sessionType.slice(index + 1);
}

export function parseRemoteAgentHostSessionTypeAuthority(sessionType: string, agentProvider: string): string | undefined {
	if (!isRemoteAgentHostSessionType(sessionType)) {
		return undefined;
	}
	const authority = findRemoteAgentHostSessionTypeAuthority(sessionType, [agentProvider]);
	return authority;
}
