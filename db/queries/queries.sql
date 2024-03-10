-- name: GetTextsByFile :many
select * from pdf_text
where filepath = ?;

-- name: GetTextByFileAndPage :one
select * from pdf_text
where filepath = ? and page = ?
order by page;

-- name: DeleteTextsByFile :exec
delete from pdf_text
where filepath = ?;

-- name: CreateText :one
insert into pdf_text (filepath, searchable_content, original_content, page, date_created)
values (?, ?, ?, ?, datetime('now'))
returning *;

-- name: GetIndexedFiles :many
select distinct filepath from pdf_text;

-- name: IsFileIndexed :one
select exists (select 1 from pdf_text where filepath = ?);

-- name: SearchTextsByFile :many
select * from pdf_text
where filepath = ? and searchable_content like ?
order by page;

-- name: GetOriginalTextsByFile :many
select page, original_content from pdf_text
where filepath = ? order by page;

