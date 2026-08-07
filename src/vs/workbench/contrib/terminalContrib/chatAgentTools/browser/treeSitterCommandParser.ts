/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

export const enum TreeSitterCommandParserLanguage {
	Bash = 'bash',
	PowerShell = 'powershell'
}

export class TreeSitterCommandParser {
	async extractSubCommands(_language: TreeSitterCommandParserLanguage, _commandLine: string): Promise<string[]> {
		return [];
	}
}
