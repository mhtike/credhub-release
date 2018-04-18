#!/usr/bin/env bash

export PATH=/var/vcap/bosh/bin:$PATH

<% port = p('credhub.port') %>

curl "https://localhost:<%= port %>/management" -X POST -d '{"read_only_mode":"true"}' -H 'content-type: application/json' -k
exec /var/vcap/jobs/credhub/bin/bbr/wait-for-stop
