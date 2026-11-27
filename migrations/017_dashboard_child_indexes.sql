CREATE INDEX dashboard_tabs_by_board_position ON dashboard_tabs(dashboard_id, position);
CREATE INDEX dashboard_cards_by_board ON dashboard_cards(dashboard_id);
CREATE INDEX dashboard_filters_by_board ON dashboard_filters(dashboard_id);
