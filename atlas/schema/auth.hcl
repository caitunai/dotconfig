table "users" {
  schema = schema.default
  column "id" {
    null = false
    type = bigint
    auto_increment = true
  }
  column "name" {
    null = false
    type = varchar(100)
  }
  primary_key {
    columns = [column.id]
  }
}

table "roles" {
  schema = schema.default
  column "id" {
    null = false
    type = bigint
    auto_increment = true
  }
  column "name" {
    null = false
    type = varchar(100)
  }
  primary_key {
    columns = [column.id]
  }
}

table "permissions" {
  schema = schema.default
  column "id" {
    null = false
    type = bigint
    auto_increment = true
  }
  column "name" {
    null = false
    type = varchar(100)
  }
  column "created_at" {
    null = false
    type = timestamp
    default = sql("CURRENT_TIMESTAMP")
  }
  column "updated_at" {
    null = false
    type = timestamp
    default = sql("CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP")
  }
  column "deleted_at" {
    null = true
    type = timestamp
  }
  primary_key {
    columns = [column.id]
  }
}
