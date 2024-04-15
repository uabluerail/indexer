$current[0].dashboard as $cur
| ([$cur | .. | select(.datasource?.type? == "prometheus")] | first | .datasource.uid) as $datasource

| .templating.list = [
  .templating.list[] | .name as $name
  | .current = ($cur.templating.list[] | select(.name == $name) | .current) // {}
]

| . as $dash

| [paths(type == "object"
    and .datasource.type? == "prometheus")] as $uids

| reduce $uids[] as $path ($dash; setpath($path + ["datasource", "uid"]; $datasource))

| .id = $cur.id
| .version = $cur.version
| {dashboard: ., overwrite: false}
