insert into pds (host, disabled) values
 ('https://agaric.us-west.host.bsky.network', FALSE),
 ('https://amanita.us-east.host.bsky.network', FALSE),
 ('https://blewit.us-west.host.bsky.network', FALSE),
 ('https://boletus.us-west.host.bsky.network', FALSE),
 ('https://chaga.us-west.host.bsky.network', FALSE),
 ('https://conocybe.us-west.host.bsky.network', FALSE),
 ('https://enoki.us-east.host.bsky.network', FALSE),
 ('https://hydnum.us-west.host.bsky.network', FALSE),
 ('https://inkcap.us-east.host.bsky.network', FALSE),
 ('https://lepista.us-west.host.bsky.network', FALSE),
 ('https://lionsmane.us-east.host.bsky.network', FALSE),
 ('https://maitake.us-west.host.bsky.network', FALSE),
 ('https://morel.us-east.host.bsky.network', FALSE),
 ('https://oyster.us-east.host.bsky.network', FALSE),
 ('https://porcini.us-east.host.bsky.network', FALSE),
 ('https://puffball.us-east.host.bsky.network', FALSE),
 ('https://russula.us-west.host.bsky.network', FALSE),
 ('https://shiitake.us-east.host.bsky.network', FALSE),
 ('https://shimeji.us-east.host.bsky.network', FALSE),
 ('https://verpa.us-west.host.bsky.network', FALSE)
on conflict do nothing;
