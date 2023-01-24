# GraphQL Limits

[Traefik](https://github.com/traefik/traefik) Middleware which allows filtering GraphQL queries by different limits

## Options

`GraphQLPath`

*Optional, Default: /graphql*

Controls which POST requests to check for GraphQL queries

`DepthLimit`

*Optional, Default: 0*

Check if the query depth does not exceed the limit. We count depth as each selection set, excluding top-level

`BatchLimit`

*Optional, Default: 0*

Check if the query does not have more batches than limit

`NodeLimit`

*Optional, Default: 0*

Check if query total number of nodes does not exceed the limit. We defined node as a selection set excluding top-level wrappers

## Configuration


### Static

```yaml
pilot:
  token: xxx
experimental:
  plugins:
    traefik-graphql-limits:
      modulename: github.com/narrative-bi/traefik-graphql-limits
      version: v0.1.0
```

### Dynamic

```yaml
http:
  routers:
    graphql-server-entrypoint:
      service: graphql-server-service
      entrypoints:
        - graphql-server-entrypoint
      rule: Host(`localhost`)
      middlewares:
        - my-traefik-graphql-limits

  services:
    graphql-server-service:
      loadBalancer:
        servers:
          - url: http://localhost:5000/

  middlewares:
    my-traefik-graphql-limits:
      plugin:
        traefik-graphql-limits:
          GraphQLPath: /graphql
          DepthLimit: 5
          BatchLimit: 2
          NodeLimit: 25
```

### Testing

You can run tests by running `make`
