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
// colors uses the Foundry palette shared by sibling benchmark reports.
const colors = { smtp:'#FF5E3A',resend:'#7D6EE7',postmark:'#FFB454',mailgun:'#FF6B85',sendgrid:'#5FCFA8',ses:'#FFC24D',log:'#B58AE8',fake:'#A9A1B3' }
const number = value => value.toLocaleString('en-US', { maximumFractionDigits: 2 })
const metrics = [
 { file:'benchmarks_ns.svg',title:'Mail Client Send Latency',label:'Send return latency',value:row=>row.time,format:time,higher:false },
 { file:'benchmarks_ops.svg',title:'Mail Client Send Throughput',label:'Serial send throughput',value:row=>1e9/row.time,format:value=>`${number(value)} sends/s`,higher:true },
 { file:'benchmarks_bytes.svg',title:'Mail Client Allocated Bytes',label:'Allocated bytes',value:row=>row.bytes,format:value=>`${number(value)} B/op`,higher:false },
 { file:'benchmarks_allocs.svg',title:'Mail Client Allocations',label:'Allocations',value:row=>row.allocs,format:value=>`${number(value)} allocs/op`,higher:false },
]
const xml = value => String(value).replace(/[&<>"']/g, character=>({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&apos;' })[character])

// renderChart mirrors the horizontal SVG layout in Events and Storage reports.
const renderChart = metric => {
 const sorted = [...rows].sort((a,b)=>(metric.higher?-1:1)*(metric.value(a)-metric.value(b)) || a.driver.localeCompare(b.driver))
 const width=1200,left=170,right=100,top=92,barHeight=28,rowGap=48
 const height=top+barHeight+(sorted.length-1)*rowGap+28,chartWidth=width-left-right
 const max=Math.max(...sorted.map(metric.value)) || 1
 const svg = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="title description">`,
  `<title id="title">${xml(metric.title)}</title>`,
  '<desc id="description">Medians of seven local fixture samples per driver. Linear bars. Client costs only, not provider latency or inbox delivery.</desc>',
  '<rect width="100%" height="100%" rx="8" fill="#1A1620"/>',
  `<text x="600" y="32" text-anchor="middle" fill="#FFFFFF" font-size="26" font-weight="700" letter-spacing="-.6" font-family="Space Grotesk, Inter, system-ui, sans-serif">${xml(metric.title)}</text>`,
  `<text x="600" y="60" text-anchor="middle" fill="#A9A1B3" font-size="16" font-family="Inter, system-ui, sans-serif">Local acceptance fixtures · median of 7 samples · linear scale · ${metric.higher?'higher':'lower'} is better</text>`,
 ]
 for (const [index,row] of sorted.entries()) {
  const y=top+index*rowGap,barWidth=Math.max(4,Math.round(metric.value(row)/max*chartWidth))
  const inside=barWidth>chartWidth-210,valueX=left+barWidth+(inside?-16:16)
  svg.push(
   `<text x="${left-16}" y="${y+20}" text-anchor="end" fill="#A9A1B3" font-size="20" font-family="Inter, system-ui, sans-serif">${xml(names[row.driver])}</text>`,
   `<rect x="${left}" y="${y}" width="${chartWidth}" height="${barHeight}" rx="5" fill="#2C2734" stroke="#3D3349"/>`,
   `<rect x="${left}" y="${y}" width="${barWidth}" height="${barHeight}" rx="5" fill="${colors[row.driver]}"/>`,
   `<text x="${valueX}" y="${y+20}" text-anchor="${inside?'end':'start'}" fill="${inside?'#08070A':'#FFFFFF'}" font-size="18" font-family="JetBrains Mono, ui-monospace, monospace">${xml(metric.format(metric.value(row)))}</text>`,
  )
 }
 return [...svg,'</svg>',''].join('\n')
}
const block = `<!-- benchmarks:start -->
Median of seven one-second samples, ${data.metadata.go_version.replace('go version ','')}, ${data.metadata.cpu}, GOMAXPROCS=1. One recipient, text and HTML bodies, and a 4 KiB attachment.

These are local client costs, not a ranking of real provider latency or inbox delivery. API drivers use HTTP acceptance fixtures in separate processes without TLS; SMTP uses local Mailpit. Log serializes to \`io.Discard\`; Fake resets every 128 sends, with reset cost included. Fixture-server allocations stay outside the client measurements.

${metrics.map(metric=>`### ${metric.label}\n\n![${metric.title} chart](docs/bench/${metric.file})`).join('\n\n')}

Throughput is the reciprocal of median send latency for a serial client, not a concurrent load test. All charts use linear bars; very small values have a minimum visible width. Driver colors stay consistent across metrics.

<details>
<summary>Recorded medians</summary>

| Driver | Send return | Bytes/op | Allocs/op |
| :-- | --: | --: | --: |
${rows.map(row=>`| ${names[row.driver]} | ${time(row.time)} | ${row.bytes.toLocaleString('en-US')} | ${row.allocs.toLocaleString('en-US')} |`).join('\n')}

</details>

[Raw samples](docs/bench/benchmarks_rows.json) · [Original Go output](docs/bench/benchmarks.txt) · [Workload and reproduction](docs/bench/README.md)
<!-- benchmarks:end -->`
const file = path.resolve(root,'../../README.md')
const current = fs.readFileSync(file,'utf8')
if(!current.includes('<!-- benchmarks:start -->')||!current.includes('<!-- benchmarks:end -->')) throw new Error('README benchmark markers are missing')
const expected = current.replace(/<!-- benchmarks:start -->[\s\S]*?<!-- benchmarks:end -->/,block)
if(process.argv.includes('--check')) {
 if(current!==expected) throw new Error('README benchmarks are stale: run node docs/bench/render.mjs')
 for (const metric of metrics) {
  if(fs.readFileSync(path.join(root,metric.file),'utf8')!==renderChart(metric)) throw new Error(`${metric.file} is stale: run node docs/bench/render.mjs`)
 }
 console.log('README benchmarks and all four SVG charts match the recorded samples.')
} else {
 fs.writeFileSync(file,expected)
 for (const metric of metrics) fs.writeFileSync(path.join(root,metric.file),renderChart(metric))
}
