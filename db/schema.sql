create table if not exists pdf_text (
    filepath text not null,
    page integer not null,
    searchable_content text not null,
    original_content text not null,
    date_created text not null
);

create index if not exists pdf_text_filepath_idx on pdf_text (filepath);