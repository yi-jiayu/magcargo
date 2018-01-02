# Magcargo
A simple self-hosted URL shortener

## Usage
Magcargo uses a HTTP interface for adding and visiting shortened links.

### Shortening a new link
```
POST /
Content-Type: application/x-www-form-urlencoded

url=https%3A%2F%2Fgithub.com%2Fyi-jiayu%2Fmagcargo
```

```
201 Created

qjrX6
```
### Visiting a shortened link
```
GET /qjrX6

```

```
303 See Other
Location: https://github.com/yi-jiayu/magcargo

<a href="https://github.com/yi-jiayu/magcargo">See Other</a>.
```

## Todo
- [ ] Remove hardcoded salt
- [ ] CLI client
- [ ] Link statistics
