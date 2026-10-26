resource "aws_vpc" "principal" {
  cidr_block           = var.cidr_vpc
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = { Name = local.nom }
}

resource "aws_default_security_group" "defaut" {
  vpc_id = aws_vpc.principal.id

  tags = { Name = "${local.nom}-defaut-vide" }
}

resource "aws_internet_gateway" "principal" {
  vpc_id = aws_vpc.principal.id

  tags = { Name = local.nom }
}

resource "aws_subnet" "public" {
  count = 2

  vpc_id                  = aws_vpc.principal.id
  availability_zone       = var.zones[count.index]
  cidr_block              = local.cidr_publics[count.index]
  map_public_ip_on_launch = false

  tags = { Name = "${local.nom}-public-${var.zones[count.index]}" }
}

resource "aws_subnet" "prive" {
  count = 2

  vpc_id            = aws_vpc.principal.id
  availability_zone = var.zones[count.index]
  cidr_block        = local.cidr_prives[count.index]

  tags = { Name = "${local.nom}-prive-${var.zones[count.index]}" }
}

resource "aws_eip" "nat" {
  domain = "vpc"

  tags = { Name = "${local.nom}-nat" }
}

resource "aws_nat_gateway" "unique" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[0].id

  tags = { Name = local.nom }

  depends_on = [aws_internet_gateway.principal]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.principal.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.principal.id
  }

  tags = { Name = "${local.nom}-public" }
}

resource "aws_route_table" "prive" {
  vpc_id = aws_vpc.principal.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.unique.id
  }

  tags = { Name = "${local.nom}-prive" }
}

resource "aws_route_table_association" "public" {
  count = 2

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "prive" {
  count = 2

  subnet_id      = aws_subnet.prive[count.index].id
  route_table_id = aws_route_table.prive.id
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = aws_vpc.principal.id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_route_table.prive.id]

  tags = { Name = "${local.nom}-s3" }
}
