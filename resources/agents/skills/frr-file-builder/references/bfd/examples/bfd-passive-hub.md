# Hub Of A Star Network

This router is the central node of a star network, and the clients on its links to the spokes
bind BFD there, each naming `profile HUB` on its Binding line. A passive session does not
attempt to start the connection: it waits for control packets from the peer before it begins
replying, so the hub sends no control packets it does not need. Each spoke keeps the default,
active mode, and starts its session. The request also wants the hub to log each session going up
and coming down from up.

A reference for shape. Copy the structure, never the values or names.

```
bfd
 profile HUB
  passive-mode
  log-session-changes
 exit
exit
!
```

What the shape demonstrates:

- `passive-mode` in the profile of the hub only, with the spokes left active
- `log-session-changes` written because the request asks for the state changes in the log

Sources: bfd › Peer / Profile Configuration.
