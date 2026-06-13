export const CONFIG_FILENAME = 'config.yaml'

/** Mocked hump configuration shown in the read-only source viewer. */
export const MOCK_CONFIG_YAML = `sinks:
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
`
