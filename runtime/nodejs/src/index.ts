// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

import * as crypto from 'crypto'
import * as proto from 'elkoprotocol'
import * as highwayhash from 'highwayhash'
import * as Long from 'Long'
import * as net from 'net'
import {PromiseSocket} from 'promise-socket'

const opcode = {
	ClientHeartbeat: 1,
	ClientHello: 2,
	ClientRequest: 3,
	ClientResponse: 4,
	ClientShutdown: 5,
	ServerHello: 1,
	ServerRequest: 2,
	ServerShutdown: 3,
}

export class Context {
	id: string

	constructor(id: string) {
		this.id = id
	}

	withTimeout(duration: number) {}
}

class Queue<T> {
	data: Array<T>
	hasData: boolean
	resolve?: (val: T) => void

	constructor() {
		this.data = []
	}

	pop() {
		return new Promise(resolve => {
			if (this.data.length) {
				resolve(this.data.shift())
			}
			this.resolve = resolve
		})
	}

	push(val: T) {
		this.data.push(val)
	}
}

let client: PromiseSocket<net.Socket>
let key: Buffer
let incoming = new Queue()
let outgoing = new Queue()

function sleep(duration: number) {
	return new Promise(resolve => setTimeout(resolve, duration))
}

async function write(op: number, param: any) {
	const msg: Buffer = param.constructor.encode(param).finish()
	const idx = msg.length + 5
	const buf = Buffer.alloc(idx + 8)
	buf.writeUInt8(op, 0)
	buf.writeUInt32BE(msg.length, 1)
	msg.copy(buf, 5)
	highwayhash.asBuffer(key, buf.slice(0, idx)).copy(buf, idx)
	console.log(buf)
	await client.write(buf)
}

export function run(serviceID: string) {
	let instanceID: Long | null = null
	if (process.env.INSTANCE_ID) {
		instanceID = Long.fromString(process.env.INSTANCE_ID!)
	}
	serviceID = process.env.SERVICE_ID || serviceID
	key = crypto
		.createHash('sha256')
		.update(serviceID)
		.digest()
	const port = parseInt(process.env.ELKO_PORT || '9000', 10)
	const sock = new net.Socket()
	client = new PromiseSocket(sock)
	sock.connect(port, '127.0.0.1', async () => {
		console.log('>> Connecting to Elko ...')
		await client.write('\x01')
		write(
			opcode.ClientHello,
			proto.ClientHello.create({
				instanceID,
				serviceID,
			})
		)
		while (true) {
			const req = await outgoing.pop()
		}
	})
	sock.on('error', msg => {
		console.log('!! ERROR:', msg)
		process.exit(1)
	})
	sock.on('close', () => {
		console.log('>> Connection to Elko has been closed. Exiting process ...')
		process.exit(0)
	})
	process.on('unhandledRejection', err => {
		console.log('!! ERROR: Unhandled promise rejection:', err)
		process.exit(1)
	})
}
