/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { ChildProcess, spawn } from 'child_process';
import { ILogService } from '../../../../platform/log/common/log.js';
import { GODE_ENGINE_PORT } from '../common/godeProtocol.js';

let engineProcess: ChildProcess | undefined;

/**
 * Spawns the gode-engine offscreen rendering process. The engine listens on a
 * fixed local port; the renderer (GodeView) connects to it over WebSocket.
 *
 * The engine path is resolved from the GODE_ENGINE_PATH environment variable.
 * Returns false when the engine could not be started.
 */
export function startGodeEngine(logService: ILogService): boolean {
	try {
		const envPath = process.env.GODE_ENGINE_PATH;
		if (!envPath) {
			logService.info('[gode] GODE_ENGINE_PATH not set; gode-engine not started');
			return false;
		}

		engineProcess = spawn(envPath, ['--port', String(GODE_ENGINE_PORT)], {
			stdio: ['ignore', 'pipe', 'pipe']
		});
		engineProcess.stdout?.on('data', (data: Buffer) => {
			const line = data.toString().trim();
			if (line) {
				logService.info(`[gode-engine] ${line}`);
			}
		});
		engineProcess.stderr?.on('data', (data: Buffer) => {
			const line = data.toString().trim();
			if (line) {
				logService.error(`[gode-engine] ${line}`);
			}
		});
		engineProcess.on('exit', (code) => logService.info(`[gode-engine] exited with code ${code}`));
		engineProcess.on('error', (err) => logService.error(`[gode-engine] error: ${err}`));

		logService.info(`[gode] gode-engine spawned on port ${GODE_ENGINE_PORT}`);
		return true;
	} catch (err) {
		logService.error(`[gode] failed to spawn gode-engine: ${err}`);
		return false;
	}
}

export function stopGodeEngine(): void {
	if (engineProcess) {
		engineProcess.kill();
		engineProcess = undefined;
	}
}
