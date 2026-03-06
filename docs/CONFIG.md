# Configuration

I want to create config parser.
Config file is yaml with two sections - `sinks` and `tables`:

```yaml
sinks:
  - name: k1
    type: kafka
tables:
  - schema: public
    name: tags
    sinks:
      - preset: k1
```

Sinks are just presets while tables defines configuration of process per table.

## Sinks

`sinks` is a collection of presets. The required fields of preset is:

- `name` unique present string name
- `type` sink type (predefined list of sinks - `kafka` and `http` for now)
- the reset of the fields are specific to particular type

- `type=kafka`
  - `connection` - connection string for kafka
  - `partitionKey` - string or text/template string of partition keyin kafka
  - `payload` - string text/template for payload that being sent to kafka
- `type=http`
  - `url` - url endpoint to message send
  - `method` - HTTP method being used to send a request
  - `body` - text/template for HTTP body

Example:

```yaml
sinks:
  - name: k1
    type: kafka
    connection: "localhost:9092"
    topic: users
    partitionKey: "{{ .id }}"
    payload: |
      {{ .Values }}
  - name: http1
    type: http
    url: "http://localhost:8080/users"
    method: POST
    body: |
      {
        "id": {{ .id }},
        "name": "{{ .name }}"
      }
```

Sinks itself does not do any work, it just preset collection that could be references from `tables`.

## Tables

`tables` is a collection of table configuration that tool will listen for changes.

Each configuration if `tables` collection could reference on or few tables.

Example of one table:

```yaml
tables:
  - schema: public
    name: tags
```

Example of a few tables:

```yaml
tables:
  - tables:
      - schema: public
        name: users
      - schema: public
        name: orders
      - schema: public
        name: products
```

So `schema`/`name` or `tables` array is mandatory. Cannot define it both way.
In addition to that there is `sinks` property, which is collection of sink configuration.

Example:

```yaml
tables:
  - schema: public
    table: users
    sinks:
      - type: kafka
        connection: "localhost:9092"
        topic: users
        partitionKey: "{{ .id }}"
        payload: |
          {{ .Values }}
```

### How sink could be configured for the table(s)

Each sink configuration is configured the same way as in "Sinks" section - exactly the same rules.

In addition to that when user configures sink for table, he can add `preset` key.

The `preset` key is just a reference to presets in `sinks` section, so `preset` key is sink name in `sinks`.

In minimal case user can only define `preset` and nothing else - all params would be inhiretend from sink preset.

```yaml
tables:
  - schema: public
    name: tags
    sinks:
      - preset: http1
      - preset: k1
```

Also, user can override some of the fields from preset.

```yaml
sinks:
  - name: k1
    type: kafka
    connection: "localhost:9092"
    topic: users
    partitionKey: "{{ .id }}"
    payload: |
      {{ .Values }}
tables:
  - schema: public
    name: tags
    sinks:
      - preset: k1
        topic: tags
```

Example of more detailed override:

```yaml
sinks:
  - name: k1
    type: kafka
    connection: "localhost:9092"
    topic: users
    partitionKey: "{{ .id }}"
    payload: |
      {{ .Values }}
tables:
  - schema: public
    table: users
    sinks:
      - preset: k1
        topic: users
        partitionKey: "{{ .id }}"
        payload: |
          {{ .Values }}
```

The only field table configuration could not be overriden is `type`, so this incorrect:

```yaml
sinks:
  - name: k1
    type: kafka
    connection: "localhost:9092"
    topic: users
    partitionKey: "{{ .id }}"
    payload: |
      {{ .Values }}
tables:
  - schema: public
    table: users
    sinks:
      - preset: k1
        type: http
```

## Validation

I want to implement parser/validator that accepts string (yaml), checks it is correct yaml, parses sinks/tables using rules provided above, and put it into the objects so it could be used later:

```go
cfg, err := config.Parse("...")
```
