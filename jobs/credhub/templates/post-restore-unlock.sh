#!/usr/bin/env bash

set -u

export PATH=/var/vcap/bosh/bin:/var/vcap/jobs/credhub/bin:$PATH
<% port = p('credhub.port') %>

curl "https://localhost:<%= port %>/management" -X POST -d '{"read_only_mode":"false"}' -H 'content-type: application/json' -k
exec /var/vcap/jobs/credhub/bin/bbr/post-bbr-start
