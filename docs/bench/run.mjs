#!/usr/bin/env node
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import net from 'node:net'
import { fileURLToPath } from 'node:url'

const root = path.dirname(fileURLToPath(import.meta.url))
const containers = []
const env = { ...process.env, GOWORK: 'off', GOMAXPROCS: '1' }
const run = (command, args, options = {}) => {
  const result = spawnSync(command, args, { cwd: root, env, encoding: 'utf8', maxBuffer: 16 * 1024 * 1024, ...options })
  if (result.status !== 0) throw new Error(`${command} ${args[0]} failed: ${result.stderr || result.stdout || result.error}`)
  return result.stdout.trim()
}
const start = (image, port, settings) => {
  const id = run('docker', ['run', '--detach', '--rm', '--publish', `127.0.0.1::${port}`, ...settings.flatMap(value => ['--env', value]), image])
  containers.push(id)
  const address = run('docker', ['port', id, `${port}/tcp`])
  const mapped = address.split(':').at(-1)
  return { id, port: mapped, image, digest: JSON.parse(run('docker', ['image', 'inspect', image]))[0].Id }
}
const ready = async service => {
  for (let attempt=0; attempt<60; attempt++) {
    const connected = await new Promise(resolve=>{
      const socket = net.createConnection({host:'127.0.0.1',port:Number(service.port)})
      socket.setTimeout(1000)
      socket.once('connect',()=>{socket.destroy();resolve(true)})
      socket.once('error',()=>{socket.destroy();resolve(false)})
      socket.once('timeout',()=>{socket.destroy();resolve(false)})
    })
    if(connected) return
    await new Promise(resolve=>setTimeout(resolve,500))
  }
  throw new Error('Mailpit readiness timed out')
}
// libraryFiles makes source edits require fresh results as well as harness edits.
const libraryFiles = []
const collect = dir => {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && !['.git', 'docs', 'examples', 'scripts'].includes(entry.name)) collect(path.join(dir, entry.name))
    else if (entry.isFile() && (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go') || ['go.mod', 'go.sum'].includes(entry.name))) libraryFiles.push(path.relative(root, path.join(dir, entry.name)))
  }
}
collect(path.resolve(root, '../..'))
libraryFiles.sort()
const hash = file => crypto.createHash('sha256').update(fs.readFileSync(path.join(root, file))).digest('hex')
try {
  const smtp = start('axllent/mailpit:v1.30.4', 1025, ['MP_MAX_MESSAGES=100', 'MP_SMTP_AUTH_ACCEPT_ANY=true', 'MP_SMTP_AUTH_ALLOW_INSECURE=true'])
  env.FORJ_BENCH_SMTP_ADDR = `127.0.0.1:${smtp.port}`
  await ready(smtp)
  console.log('Running Mail driver benchmarks: seven 1s samples per driver, GOMAXPROCS=1.')
  const output = run('go', ['test', '-run=^$', '-bench=^BenchmarkMailSend$', '-benchmem', '-count=7', '-benchtime=1s', '-cpu=1', '-timeout=20m'])
  const samples = []
  for (const line of output.split('\n')) {
    const match = line.match(/^Benchmark(MailSend)\/(\w+)(?:-\d+)?\s+(\d+)\s+([\d.]+) ns\/op\s+([\d.]+) B\/op\s+([\d.]+) allocs\/op/)
    if (match) samples.push({ family: match[1] === 'MailSend' ? 'mail' : 'database', driver: match[2], iterations: Number(match[3]), ns_op: Number(match[4]), bytes_op: Number(match[5]), allocs_op: Number(match[6]) })
  }
  const expected = { mail: ['smtp','resend','postmark','mailgun','sendgrid','ses','log','fake'] }
  for (const [family, drivers] of Object.entries(expected)) for (const driver of drivers) {
    const rows = samples.filter(row => row.family === family && row.driver === driver)
    if (rows.length !== 7 || rows.some(row => row.ns_op <= 0)) throw new Error(`missing or invalid samples: ${family}/${driver}`)
  }
  if (samples.length !== 56) throw new Error(`unexpected sample count: ${samples.length}`)
  const results = {
    metadata: {
      captured_at: new Date().toISOString(), go_version: run('go', ['version']), goos: run('go', ['env','GOOS']), goarch: run('go',['env','GOARCH']),
      cpu: os.cpus()[0].model, gomaxprocs: 1, sample_count: 7, benchmark_time: '1s',
      library_revision: run('git', ['rev-parse', 'HEAD'], { cwd: path.resolve(root, '../..') }),
      packages: { mail: 'local repository source', mailses: 'local sibling module' },
      fixtures: [smtp].map(({image,digest})=>({image,digest})),
      harness_sha256: Object.fromEntries(['mail_test.go','fixture_test.go','go.mod','go.sum','run.mjs', ...libraryFiles].map(file=>[file,hash(file)])),
      mail: 'Client send return for one recipient, text and HTML bodies, and a 4 KiB attachment. SMTP uses local plaintext Mailpit. Provider API drivers use local HTTP acceptance fixtures in separate processes with reused connections, no TLS, and dummy credentials. SES includes SDK signing. Log writes JSON to io.Discard; Fake resets every 128 sends, with reset cost included. No real provider calls or inbox delivery.'
    }, samples
  }
  fs.writeFileSync(path.join(root,'benchmarks_rows.json'),JSON.stringify(results,null,2)+'\n')
  fs.writeFileSync(path.join(root,'benchmarks.txt'),output+'\n')
  run(process.execPath, [path.join(root, 'render.mjs')])
  console.log(`Recorded ${samples.length} samples across all eight Mail drivers.`)
} finally {
  for (const id of containers.reverse()) spawnSync('docker',['rm','--force',id],{stdio:'ignore'})
}
