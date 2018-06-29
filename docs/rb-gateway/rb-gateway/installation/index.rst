.. _rb-gateway-installation:

===============================
Installing Review Board Gateway
===============================

``rb-gateway`` is currently distributed as a standalone binary. Installation
consists of a few simple steps:

1. `Download the latest version`_ of ``rb-gateway``, place the binary somewhere
   on your server (for example, on Linux,
   :file:`/usr/local/bin/rb-gateway`), and make it executable.
2. Create a directory for ``rb-gateway`` to store authentication token and
   webhook data. On Linux, a good place for this would be
   :file:`/var/lib/rb-gateway`.
3. Create the :ref:`configuration file <rb-gateway-configuration>`.
4. Set up ``rb-gateway`` to run :ref:`as a service <rb-gateway-service>`.


.. _Download the latest version: https://www.reviewboard.org/downloads/rbgateway/



For example, to get started quickly on Linux::

    $ sudo curl https://www.reviewboard.org/downloads/rbgateway/latest/linux_amd64/ \
           -O /usr/local/bin/rb-gateway
    $ sudo chmod +x /usr/local/bin/rb-gateway
    $ sudo mkdir /var/lib/rb-gateway
    $ sudo mkdir /etc/rb-gateway
    $ sudo vim /etc/rb-gateway/rb-gateway.conf
