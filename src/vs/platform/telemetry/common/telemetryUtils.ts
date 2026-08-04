/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ITelemetryService, TelemetryLevel } from './telemetry.js';

/**
 * A special class used to denoting a telemetry value which should not be clean.
 * This is because that value is "Trusted" not to contain identifiable information such as paths.
 */
export class TelemetryTrustedValue<T> {
	public readonly isTrustedTelemetryValue = true;
	constructor(public readonly value: T) { }
}

export class NullTelemetryServiceShape implements ITelemetryService {
	declare readonly _serviceBrand: undefined;
	readonly telemetryLevel = TelemetryLevel.NONE;
	readonly sessionId = 'someValue.sessionId';
	readonly machineId = 'someValue.machineId';
	readonly sqmId = 'someValue.sqmId';
	readonly devDeviceId = 'someValue.devDeviceId';
	readonly firstSessionDate = 'someValue.firstSessionDate';
	readonly sendErrorTelemetry = false;
	publicLog() { }
	publicLog2() { }
	publicLogError() { }
	publicLogError2() { }
	setExperimentProperty() { }
	setCommonProperty() { }
}

export const NullTelemetryService = new NullTelemetryServiceShape();

export class NullEndpointTelemetryService implements ITelemetryService {
	_serviceBrand: undefined;
	readonly telemetryLevel = TelemetryLevel.NONE;
	readonly sessionId = '';
	readonly machineId = '';
	readonly sqmId = '';
	readonly devDeviceId = '';
	readonly firstSessionDate = '';
	readonly sendErrorTelemetry = false;
	publicLog() { }
	publicLog2() { }
	publicLogError() { }
	publicLogError2() { }
	setExperimentProperty() { }
	setCommonProperty() { }
}

export function supportsTelemetry(_productService: unknown, _environmentService: unknown): boolean {
	return false;
}

export function isLoggingOnly(_productService: unknown, _environmentService: unknown): boolean {
	return true;
}

export function getTelemetryLevel(_configurationService: unknown): TelemetryLevel {
	return TelemetryLevel.NONE;
}

export function isInternalTelemetry(_productService: unknown, _configurationService?: unknown): boolean {
	return false;
}

export function getPiiPathsFromEnvironment(_environmentService: unknown): string[] {
	return [];
}

export interface ITelemetryAppender {
	log(eventName: string, data?: Record<string, any>): Promise<unknown>;
	flush(): Promise<unknown>;
}

export const NullAppender: ITelemetryAppender = {
	log: () => Promise.resolve(undefined),
	flush: () => Promise.resolve(undefined),
};

export function cleanData(data: Record<string, any>, _cleanupPatterns: RegExp[]): Record<string, any> {
	return data;
}

export function removePropertiesWithPossibleUserInfo(props: Record<string, any>): Record<string, any> {
	return props;
}

export function validateTelemetryData(data: Record<string, any>): Record<string, any> {
	return data;
}

export function cleanRemoteAuthority(remoteAuthority: string, _productService?: unknown): string {
	return remoteAuthority || 'other';
}

export const TelemetryLogGroup = 'telemetry';

export function anonymizeFilePaths(_productService: unknown, stack?: string): string {
	return stack || '';
}
