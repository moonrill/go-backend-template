# Redis

Redis is an optional integration planned for a later phase. When enabled, configuration must be validated, operations must have deadlines, and the client must be closed during shutdown. Do not make Redis a requirement for core API startup unless the application explicitly depends on it.
