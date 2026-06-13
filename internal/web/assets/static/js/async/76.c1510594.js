"use strict";(self.webpackChunkhump=self.webpackChunkhump||[]).push([["76"],{2795(e,a,s){s.d(a,{E:()=>i});var t=s(2361),r=s(6085),n=s(5291),l=s(8260);let o=(0,n.F)("inline-flex min-h-6 items-center gap-1.5 border px-2 py-0.5 text-sm font-medium whitespace-nowrap",{variants:{variant:{neutral:"bg-surface-3 text-ink-muted [&_[data-slot=dot]]:bg-ink-subtle",healthy:"bg-success-soft text-success",lag:"bg-amber-soft text-amber",failed:"bg-danger-soft text-danger",active:"bg-primary-soft text-primary"},border:{true:"rounded-full",false:"rounded-md"}},compoundVariants:[{variant:"neutral",border:!1,class:"border-surface-3"},{variant:"healthy",border:!1,class:"border-success-soft"},{variant:"lag",border:!1,class:"border-amber-soft"},{variant:"failed",border:!1,class:"border-danger-soft"},{variant:"active",border:!1,class:"border-primary-soft"},{variant:"neutral",border:!0,class:"border-hairline"},{variant:"healthy",border:!0,class:"border-success/30"},{variant:"lag",border:!0,class:"border-amber/30"},{variant:"failed",border:!0,class:"border-danger/30"},{variant:"active",border:!0,class:"border-primary/30"}],defaultVariants:{variant:"neutral",border:!1}}),i=e=>{let{className:a,variant:s,border:n=!1,asChild:i=!1,dot:c=!1,dotAnimated:m=!1,icon:d,title:u,children:b,...p}=e,h=i?r.DX:"span";return(0,t.jsx)(h,{className:(0,l.cn)(o({variant:s,border:n,className:a})),...p,children:i?b:(0,t.jsxs)(t.Fragment,{children:[c&&(0,t.jsx)("span",{"data-slot":"dot",className:(0,l.cn)("size-1.5 shrink-0 rounded-full bg-current",m&&"animate-ping")}),d&&(0,t.jsx)("span",{"data-slot":"icon",className:"inline-flex shrink-0 opacity-60 [&_svg]:size-3.5",children:d}),u&&(0,t.jsx)("span",{"data-slot":"title",className:"opacity-60",children:u}),b]})})}},1735(e,a,s){let t;s.r(a),s.d(a,{component:()=>v});var r=s(2361),n=s(8413),l=s(6387),o=s(6804),i=s(7363),c=s(2795),m=s(6102),d=s(7113),u=s(946),b=s(6851),p=s(8260);let h={name:"hump-code",type:"light",bg:"var(--syntax-bg)",fg:"var(--syntax-fg)",settings:[{settings:{background:"var(--syntax-bg)",foreground:"var(--syntax-fg)"}},{scope:["entity.name.tag","meta.tag","support.type.property-name"],settings:{foreground:"var(--syntax-tag)"}},{scope:["string"],settings:{foreground:"var(--syntax-string)"}},{scope:["constant.numeric"],settings:{foreground:"var(--syntax-number)"}},{scope:["constant.language","keyword"],settings:{foreground:"var(--syntax-keyword)"}},{scope:["comment"],settings:{foreground:"var(--syntax-comment)",fontStyle:"italic"}},{scope:["punctuation","keyword.operator","entity.name.tag.anchor","storage.type"],settings:{foreground:"var(--syntax-punctuation)"}}]},g=e=>{let{value:a,className:s}=e,[l,o]=(0,n.useState)(null);return(0,n.useEffect)(()=>{let e=!0;return(t??(t=(0,u.tz)({themes:[h],langs:[d.A],engine:(0,b.l)()})),t).then(s=>{e&&o(s.codeToHtml(a,{lang:"yaml",theme:"hump-code"}))}),()=>{e=!1}},[a]),(0,r.jsx)("div",{className:(0,p.cn)("hump-code min-h-0 flex-1 overflow-auto",s),children:l?(0,r.jsx)("div",{dangerouslySetInnerHTML:{__html:l}}):(0,r.jsx)("pre",{className:"hump-code-fallback",children:a})})},f=`sinks:
    - name: kafka1
      kafka:
          url: "localhost:9092"
          topic: "{{.Table}}"
          partitionKey: "{{coalesce .Next.id .Prev.id}}"
    - name: nats-core
      nats-core:
          url: "nats://localhost:4222"
          username: "hump"
          password: "hump"
          subject: "hump.{{.Schema}}.{{.Table}}"
    - name: nats-jetstream
      nats-jetstream:
          url: "nats://localhost:4222"
          username: "hump"
          password: "hump"
          subject: "hump.js.{{.Schema}}.{{.Table}}"
          stream:
              name: HUMP
              subjects:
                  - "hump.js.>"
          retention: limits
          storage: file
    - name: rabbitmq1
      rabbitmq:
          url: "amqp://hump:hump@localhost:5672/"
          exchange:
              name: "hump.{{.Schema}}"
              type: topic
          routingKey: "{{.Schema}}.{{.Table}}"
    - name: rabbitmq-stream
      rabbitmq-stream:
          url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"
          stream:
              name: "hump.{{.Schema}}.{{.Table}}"

tables:
    - tables:
          - schema: public
            table: users
          - schema: public
            table: posts
          - schema: public
            table: comments
          - schema: public
            table: events
      sinks:
          - preset: kafka1
          - preset: nats-core
          - preset: nats-jetstream
          - preset: rabbitmq1
          - preset: rabbitmq-stream
`,x=()=>{let{copied:e,copy:a}=(0,p.iD)(),s=(0,n.useCallback)(()=>{a(f)},[a]);return(0,r.jsxs)("div",{className:"flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-hairline bg-surface-1",children:[(0,r.jsxs)("div",{className:"flex items-center justify-between border-b border-hairline px-3 py-2",children:[(0,r.jsxs)("div",{className:"flex items-center gap-2.5",children:[(0,r.jsx)(l.A,{className:"size-4 text-ink-subtle"}),(0,r.jsx)("span",{className:"font-mono text-sm text-ink",children:"config.yaml"}),(0,r.jsx)(c.E,{variant:"neutral",children:"read-only"})]}),(0,r.jsx)(m.$,{variant:"ghost",className:"size-8.5 px-0","aria-label":"Copy configuration",onClick:s,children:e?(0,r.jsx)(o.A,{className:"text-success"}):(0,r.jsx)(i.A,{})})]}),(0,r.jsx)(g,{value:f})]})},v=()=>(0,r.jsx)("div",{className:"flex h-[calc(100svh-3.5rem)] flex-col p-6",children:(0,r.jsx)(x,{})})}}]);