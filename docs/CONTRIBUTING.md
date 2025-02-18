# App structure

To keep things tidy the following approach is used:
  * `views` host and manage `panes` , and `panes` host and manage `components`
  * elements are supposed to be isolated and to communicate using bubbletea events

![application components scheme](./images/app-scheme.png "app-scheme")
