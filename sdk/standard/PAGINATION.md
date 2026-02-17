# Pagination Helper

The `pagination.go` file is a hand-written helper that deserializes the JSON-encoded
`X-Pagination` response header returned by list endpoints.

The API returns pagination metadata as a JSON string inside the `X-Pagination` header
rather than in the response body. The OpenAPI generator treats response headers as plain
strings and does not generate typed deserialization code for them, so this helper bridges
that gap.

## Future State

The preferred long-term approach is to move pagination metadata into the response body
using an envelope pattern, for example:

    {
      "data": [...],
      "pagination": {
        "pageNumber": 1,
        "pageSize": 20,
        "total": 30,
        "orderBy": "CREATED_DESC"
      }
    }

With pagination in the body, the OpenAPI spec can define a proper schema that the
generator will produce typed structs and deserialization code for automatically. This
would eliminate the need for the hand-written helper entirely.
