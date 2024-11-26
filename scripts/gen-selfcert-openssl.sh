#!/bin/bash

# >> Sertifikat yang dibuat hanya sekedar dummy cert
# >> dan digunakan hanya untuk simulasi..

# >> Generate CA untuk RS JAKUT
openssl genrsa -out ca-jakut.key 2048
openssl req -x509 -new -nodes -key ca-jakut.key -sha256 -days 3650 -out secret-pubrsjakut.crt -subj "/CN=RS David Jakarta Utara CA/O=RS David Jakarta Utara/OU=IT Security/L=Jakarta/ST=DKI Jakarta/C=ID"

# >> Generate CA untuk RS JAKPUS
echo ">> Membuat CA untuk RS JAKPUS..."
openssl genrsa -out ca-jakpus.key 2048
openssl req -x509 -new -nodes -key ca-jakpus.key -sha256 -days 3650 -out secret-pubrsjakpus.crt -subj "/CN=RS David Jakarta Pusat CA/O=RS David Jakarta Pusat/OU=IT Security/L=Jakarta/ST=DKI Jakarta/C=ID"

# >> Generate key dan csr untuk RS JAKUT
openssl genrsa -out secret-rsjakut.key 2048
openssl req -new -key secret-rsjakut.key -out secret-rsjakut.csr -subj "/CN=RS David Jakarta Utara/O=RS David Jakarta Utara/OU=IT Security/L=Jakarta Utara/ST=DKI Jakarta/C=ID"

# >> Generate key dan csr untuk RS JAKPUS
openssl genrsa -out secret-rsjakpus.key 2048
openssl req -new -key secret-rsjakpus.key -out secret-rsjakpus.csr -subj "/CN=RS David Jakarta Pusat/O=RS David Jakarta Pusat/OU=IT Security/L=Jakarta Pusat/ST=DKI Jakarta/C=ID"

# >> Generate Cert dengan csr dan key yang telah di buat sbelumnya serta sign dengan ca untuk RS JAKUT
openssl x509 -req -in secret-rsjakut.csr -CA secret-pubrsjakut.crt -CAkey ca-jakut.key -CAcreateserial -out secret-rsjakut.crt -days 365 -sha256

# >> Generate Cert dengan csr dan key yang telah di buat sbelumnya serta sign dengan ca untuk RS JAKPUS
openssl x509 -req -in secret-rsjakpus.csr -CA secret-pubrsjakpus.crt -CAkey ca-jakpus.key -CAcreateserial -out secret-rsjakpus.crt -days 365 -sha256