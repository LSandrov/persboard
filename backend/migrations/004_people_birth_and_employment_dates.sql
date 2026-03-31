alter table if exists people
  add column if not exists birth_date date,
  add column if not exists employment_date date;
