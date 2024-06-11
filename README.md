# webserv

Thin web server stub.

Given a listen address, certificate directory, user name and data directory:

* If certificate directory is not blank, reads `fullchain.pem` and `privkey.pem` from it.
* If the listen address does not specify a port, default port depends on initial user privileges and if we have a certificate.
* Starts listening on the address and port.
* If user name is given, switch to that user.
* If data directory is given, create it if needed and then switch current directory to it.
