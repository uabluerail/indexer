.dashboard | . as $dash

| [paths(type == "object"
    and (.datasource?.uid? | type) == "string"
    and .datasource.type? == "prometheus")] as $uids

| reduce $uids[] as $path ([]; ($dash | getpath($path).datasource.uid) as $uid | if [.[] == $uid] | any then . else . + [$uid] end)
| . as $unique_uids

| [range($unique_uids | length) | {key: $unique_uids[.], value: "DS\(.+1)"}]
| from_entries as $uid_map

| reduce $uids[] as $path ($dash; setpath($path + ["datasource", "uid"]; "${\($uid_map[getpath($path).datasource.uid])}"))

| reduce paths(type == "object" and has("current") and has("datasource"))
    as $path (.; setpath($path + ["current"]; {}))

| .id = null
| .__inputs = [$unique_uids[] | {
    name: $uid_map[.],
    label: "Prometheus",
    description: "",
    type: "datasource",
    pluginId: "prometheus",
    pluginName: "Prometheus",
}]
| .__requires = []
| .__elements = {}
