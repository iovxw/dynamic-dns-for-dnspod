dynamic-dns-for-dnspod
======================

DNSPod动态DNS设置工具

功能
----

自动将相应域名解析到本机IP

如果IP产生变更会自动更新

如果域名没添加会自动添加域名到账户下（当然域名NS得提前设置好）

如果子域名不存在会自动添加子域名

Quick Start
-----------

首先将域名添加到DNSPod（也可使用本程序自动添加，但是一定要先设置好NS）

然后编辑`config.yaml`，修改相应配置

开启本程序即可

下载
----

[![Gobuild Download](http://gobuild.io/badge/github.com/Bluek404/dynamic-dns-for-dnspod/downloads.svg)](http://gobuild.io/github.com/Bluek404/dynamic-dns-for-dnspod)