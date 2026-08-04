/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../../../../platform/instantiation/common/instantiation.js';

export interface IAiEditTelemetryService {
	readonly _serviceBrand: undefined;
	createSuggestionId(data: Record<string, unknown>): string;
	handleCodeAccepted(data: Record<string, unknown>): void;
	handleCodeRejected(data: Record<string, unknown>): void;
}

export const IAiEditTelemetryService = createDecorator<IAiEditTelemetryService>('aiEditTelemetryService');

export class NullAiEditTelemetryService implements IAiEditTelemetryService {
	declare readonly _serviceBrand: undefined;
	createSuggestionId(_data: Record<string, unknown>): string { return ''; }
	handleCodeAccepted(_data: Record<string, unknown>): void { }
	handleCodeRejected(_data: Record<string, unknown>): void { }
}
