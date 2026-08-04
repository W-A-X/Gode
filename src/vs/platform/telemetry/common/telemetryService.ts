/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { IConfigurationService } from '../../configuration/common/configuration.js';
import { IProductService } from '../../product/common/productService.js';
import { ITelemetryService, TelemetryLevel } from './telemetry.js';
import { ITelemetryAppender } from './telemetryUtils.js';

export interface ITelemetryServiceConfig {
	appenders: ITelemetryAppender[];
	sendErrorTelemetry?: boolean;
	commonProperties?: Record<string, any>;
	piiPaths?: string[];
	waitForExperimentProperties?: boolean;
	meteredConnectionService?: unknown;
}

/**
 * No-op telemetry service — 所有遥测数据不再发送。
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

	constructor(
		_config?: ITelemetryServiceConfig,
		_configurationService?: IConfigurationService,
		_productService?: IProductService
	) { }

	publicLog() { }
	publicLog2() { }
	publicLogError() { }
	publicLogError2() { }
	setExperimentProperty() { }
	setCommonProperty() { }
}
