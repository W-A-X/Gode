/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ITelemetryService, ITelemetryData, TelemetryLevel } from '../../../../platform/telemetry/common/telemetry.js';
import { InstantiationType, registerSingleton } from '../../../../platform/instantiation/common/extensions.js';
import { ClassifiedEvent, StrictPropertyCheck, OmitMetadata, IGDPRProperty } from '../../../../platform/telemetry/common/gdprTypings.js';

/**
 * No-op telemetry service — all遥测数据不再发送到 Microsoft。
 * 保留接口以兼容现有代码调用。
 */
export class TelemetryService implements ITelemetryService {

	declare readonly _serviceBrand: undefined;

	readonly telemetryLevel = TelemetryLevel.NONE;
	readonly sessionId = '';
	readonly machineId = '';
	readonly sqmId = '';
	readonly devDeviceId = '';
	readonly firstSessionDate = '';
	readonly msftInternal: boolean | undefined = undefined;
	readonly sendErrorTelemetry = false;

	publicLog(_eventName: string, _data?: ITelemetryData): void { }
	publicLog2<E extends ClassifiedEvent<OmitMetadata<T>> = never, T extends IGDPRProperty = never>(_eventName: string, _data?: StrictPropertyCheck<T, E>): void { }
	publicLogError(_errorEventName: string, _data?: ITelemetryData): void { }
	publicLogError2<E extends ClassifiedEvent<OmitMetadata<T>> = never, T extends IGDPRProperty = never>(_errorEventName: string, _data?: StrictPropertyCheck<T, E>): void { }
	setExperimentProperty(_name: string, _value: string): void { }
	setCommonProperty(_name: string, _value: string | boolean): void { }
}

registerSingleton(ITelemetryService, TelemetryService, InstantiationType.Delayed);
