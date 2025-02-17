terraform {
  required_providers {
    regru = {
      version = "~>0.1.0"
      source  = "sport24ru/regru"
    }
  }
}

provider "regru" {
  username = var.regru_api_username
  password = var.regru_api_password
}

resource "regru_dns_record" "www-example-com" {
  record   = "11.22.33.44"
  zone     = "example.com"
  name     = "wwww"
  type     = "A"
  priority = 0
}

resource "regru_dns_record" "test-example-com" {
  record   = "22.33.44.55"
  zone     = "example.com"
  name     = "test"
  type     = "A"
  priority = 0
}

resource "regru_dns_record" "testcase-ipv6-docs-example-com" {
  zone   = "example.com"
  name   = "testcase-ipv6"
  type   = "AAAA"
  record = "aaaa::aaaa:aaaa:aaaa:aaaa"
  priority = 0
}

resource "regru_dns_record" "wwwww-example-com" {
  record = "sport24.ru"
  zone   = "example.com"
  name   = "cname"
  type   = "CNAME"
  priority = 0
}

resource "regru_dns_record" "www-txt-example-com" {
  zone   = "example.com"
  name   = "www-txt2"
  type   = "TXT"
  record = "This is a TXT record for example.com"
  priority = 0
}

resource "regru_dns_record" "testcase-mx-example-com" {
  zone     = "example.com"
  name     = "@"
  type     = "MX"
  record   = "mail.testcase-mx.example.com"
  priority = 10
}
