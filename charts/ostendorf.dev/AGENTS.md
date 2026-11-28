# charts/ostendorf.dev

- `charts/ostendorf.dev/` is the Helm chart. Its `Chart.yaml` `name` and every rendered object
  name use `ostendorf-dev` instead of `ostendorf.dev`: a literal dot is invalid in a `Service`'s
  `metadata.name` (RFC 1035 label), which a chart named after the repository directly would
  otherwise produce by default.
