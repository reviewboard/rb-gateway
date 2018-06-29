.. _rb-gateway-index:

==================================
Review Board Gateway Documentation
==================================

The Review Board Gateway is a small service to interface your repositories with
Review Board. Git and Mercurial repository remote protocols do not always
provide the necessary APIs for Review Board For self-hosted repositories, this
has traditionally required the use of GitWeb, cgit, or HgWeb. ``rb-gateway`` is
a service with a smaller footprint that can provide the same functionality.

.. toctree::
   :maxdepth: 2

   rb-gateway/installation/index
   rb-gateway/configuration/index
