# LeetCode API

A small Go HTTP API that queries LeetCode's GraphQL API for getfolk.app.

## Run

Requires Go 1.26.5 or newer and network access to LeetCode.

```sh
go run .
```

The server listens on port `8080` when `PORT` is unset or empty. To change it:

```sh
PORT=3000 go run .
```

For deployment, build and run the executable directly so it receives shutdown signals:

```sh
go build -o leetcode-api .
./leetcode-api
```

Ctrl+C (`SIGINT`) or `SIGTERM` stops accepting new connections and gives active
requests up to 15 seconds to finish. After that, remaining connections are closed.
Startup failures and shutdown failures exit with a nonzero status. Allow at least
15 seconds for graceful shutdown in your deployment configuration.

## Endpoints

All endpoints below use `GET` and return JSON. Replace `example-user` with a
LeetCode username. Response examples are illustrative, not live data.

| Endpoint | Description | Query parameters |
| --- | --- | --- |
| `/users/{username}/exists` | Check whether a user exists | None |
| `/users/{username}/profile` | Profile and social links | None |
| `/users/{username}/stats` | Accepted and total submission statistics | None |
| `/users/{username}/submissions` | Recent submissions | `limit`: integer from 1–20; default 10 |
| `/problems/{slug}` | Problem details and plain-text description | None |

### User existence

```sh
curl 'http://localhost:8080/users/example-user/exists'
```

```json
{
  "username": "example-user",
  "exists": true
}
```

A missing user returns HTTP `200` with `"exists": false`.

### User profile

```sh
curl 'http://localhost:8080/users/example-user/profile'
```

```json
{
  "username": "example-user",
  "githubUrl": "",
  "twitterUrl": "",
  "linkedinUrl": "",
  "profile": {
    "realName": "Example User",
    "aboutMe": "Learning algorithms",
    "userAvatar": "https://example.com/avatar.png",
    "countryName": "United States",
    "company": "",
    "school": "",
    "websites": [],
    "skillTags": ["Python"],
    "ranking": 123456,
    "reputation": 0,
    "starRating": 0
  }
}
```

### User statistics

```sh
curl 'http://localhost:8080/users/example-user/stats'
```

```json
{
  "username": "example-user",
  "submitStats": {
    "acSubmissionNum": [
      {"difficulty": "All", "count": 30, "submissions": 40},
      {"difficulty": "Easy", "count": 20, "submissions": 25},
      {"difficulty": "Medium", "count": 8, "submissions": 12},
      {"difficulty": "Hard", "count": 2, "submissions": 3}
    ],
    "totalSubmissionNum": [
      {"difficulty": "All", "count": 35, "submissions": 65},
      {"difficulty": "Easy", "count": 22, "submissions": 35},
      {"difficulty": "Medium", "count": 10, "submissions": 22},
      {"difficulty": "Hard", "count": 3, "submissions": 8}
    ]
  }
}
```

### Recent submissions

```sh
curl 'http://localhost:8080/users/example-user/submissions?limit=10'
```

```json
{
  "username": "example-user",
  "recentSubmissions": [
    {
      "title": "Two Sum",
      "titleSlug": "two-sum",
      "timestamp": "1700000000",
      "statusDisplay": "Accepted",
      "lang": "python3"
    }
  ]
}
```

`limit` controls the requested maximum number of results; fewer may be available.
Omitting it or passing `?limit=` uses 10. Values outside 1–20 or non-integers
return HTTP `400`. There is no pagination parameter. `timestamp` is a Unix timestamp
in seconds encoded as a string. Results can include unsuccessful submissions.

### Problem details

```sh
curl 'http://localhost:8080/problems/two-sum'
```

```json
{
  "questionFrontendId": "1",
  "title": "Two Sum",
  "titleSlug": "two-sum",
  "difficulty": "Easy",
  "isPaidOnly": false,
  "acRate": 55.5,
  "likes": 100,
  "dislikes": 5,
  "content": "Illustrative problem description.\n\nExample input and output.",
  "topicTags": [
    {"name": "Array", "slug": "array"},
    {"name": "Hash Table", "slug": "hash-table"}
  ]
}
```

`content` is converted from HTML to plain text. `acRate` is the acceptance
percentage. Array fields may be `null` when LeetCode returns no value.

## Errors

Handler errors use this JSON format:

```json
{"error": "user not found"}
```

| Status | Meaning / example message |
| --- | --- |
| `400` | Invalid input, such as `username is required` or `limit must be an integer between 1 and 20` |
| `404` | `user not found` for profile, stats, or submissions; `problem not found` for a problem |
| `500` | Upstream request failure: `failed to connect to leetcode. try again` |

Unmatched routes and unsupported methods use Go's standard HTTP responses, which
are plain text rather than the JSON error format above.

## Tests

```sh
go test ./...
```
