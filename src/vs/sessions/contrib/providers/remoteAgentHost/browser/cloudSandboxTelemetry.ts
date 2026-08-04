/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { createDecorator } from '../../../../../platform/instantiation/common/instantiation.js';

export type CloudSandboxRequestAction = string;
export type CloudSandboxRefreshStopReason = string;

export function requestOutcomeForStatus(statusCode: number | undefined): string {
	return statusCode !== undefined ? String(statusCode) : 'unknown';
}

export interface ICloudSandboxTelemetryService {
	readonly _serviceBrand: undefined;
	reportRequest(action: string, outcome: string): void;
	reportRefreshStop(reason: string, detail: string): void;
	reportCredentialRefreshStopped(reason: string, unhealthyCycles: number, error?: unknown): void;
}

export const ICloudSandboxTelemetryService = createDecorator<ICloudSandboxTelemetryService>('cloudSandboxTelemetryService');

export class CloudSandboxTelemetryService implements ICloudSandboxTelemetryService {
	declare readonly _serviceBrand: undefined;
	reportRequest(_action: string, _outcome: string): void { }
	reportRefreshStop(_reason: string, _detail: string): void { }
	reportCredentialRefreshStopped(_reason: string, _unhealthyCycles: number, _error?: unknown): void { }
}
