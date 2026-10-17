import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.dirname(fileURLToPath(import.meta.url))
const data = JSON.parse(fs.readFileSync(path.join(root,'benchmarks_rows.json'),'utf8'))
const median = values => [...values].sort((a,b)=>a-b)[Math.floor(values.length/2)]
const names = { smtp:'SMTP',ses:'SES',resend:'Resend',postmark:'Postmark',mailgun:'Mailgun',sendgrid:'SendGrid',log:'Log',fake:'Fake' }
const rows = Object.keys(names).map(driver=>{
 const samples = data.samples.filter(row=>row.driver===driver)
 return {driver,time:median(samples.map(s=>s.ns_op)),bytes:median(samples.map(s=>s.bytes_op)),allocs:median(samples.map(s=>s.allocs_op))}
}).sort((a,b)=>a.time-b.time)
const time = value => value<1000?`${value.toFixed(2)} ns`:value<1e6?`${(value/1000).toFixed(2)} µs`:`${(value/1e6).toFixed(2)} ms`
const block = `<!-- benchmarks:start -->
Median of seven one-second samples, ${data.metadata.go_version.replace('go version ','')}, ${data.metadata.cpu}, GOMAXPROCS=1. One recipient, text and HTML bodies, and a 4 KiB attachment.

| Driver | Send return | Bytes/op | Allocs/op |
| :-- | --: | --: | --: |
${rows.map(row=>`| ${names[row.driver]} | ${time(row.time)} | ${row.bytes.toLocaleString('en-US')} | ${row.allocs.toLocaleString('en-US')} |`).join('\n')}

These are local client costs, not a ranking of real provider latency or inbox delivery. API drivers use HTTP acceptance fixtures in separate processes without TLS; SMTP uses local Mailpit. Log serializes to \`io.Discard\`; Fake resets every 128 sends, with reset cost included. Fixture-server allocations stay outside the client measurements.

[Raw samples](docs/bench/benchmarks_rows.json) · [Original Go output](docs/bench/benchmarks.txt) · [Workload and reproduction](docs/bench/README.md)
<!-- benchmarks:end -->`
const file = path.resolve(root,'../../README.md')
const current = fs.readFileSync(file,'utf8')
if(!current.includes('<!-- benchmarks:start -->')||!current.includes('<!-- benchmarks:end -->')) throw new Error('README benchmark markers are missing')
const expected = current.replace(/<!-- benchmarks:start -->[\s\S]*?<!-- benchmarks:end -->/,block)
if(process.argv.includes('--check')) {
 if(current!==expected) throw new Error('README benchmarks are stale: run node docs/bench/render.mjs')
 console.log('README benchmarks match the recorded samples.')
} else fs.writeFileSync(file,expected)
