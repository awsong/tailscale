// @ts-check
// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

import {Call} from '@wailsio/runtime';

/**
 * @function Greet
 * @param name {string}
 * @returns {Promise<string>}
 **/
export async function Greet(name) {
	return Call.ByName("main.GreetService.Greet", ...Array.prototype.slice.call(arguments, 0));
}

/**
 * @function Send
 * @param message {string}
 * @returns {Promise<void>}
 **/
export async function Send(message) {
	return Call.ByName("main.GreetService.Send", ...Array.prototype.slice.call(arguments, 0));
}
