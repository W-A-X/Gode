/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export class CommandLineAutoApprover {
	async isCommandAutoApproved(_commandLine: string, _shell: string, _os: unknown, _sessionResource: unknown): Promise<boolean> {
		return false;
	}

	isCommandLineAutoApproved(_commandLine: string): boolean {
		return false;
	}
}
