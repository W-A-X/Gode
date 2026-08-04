/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ILoggerService } from '../../log/common/log.js';
import { IProductService } from '../../product/common/productService.js';
import { ITelemetryAppender } from './telemetryUtils.js';

export class TelemetryLogAppender implements ITelemetryAppender {
	constructor(
		_channelName: string,
		_useLogs: boolean,
		_loggerService: ILoggerService,
		_environmentService: unknown,
		_productService: IProductService
	) { }

	log(_eventName: string, _data?: Record<string, any>): Promise<unknown> {
		return Promise.resolve(undefined);
	}

	flush(): Promise<unknown> {
		return Promise.resolve(undefined);
	}
}
