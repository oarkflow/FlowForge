# ==============================================================================
# Laravel PHP — Docker Image
# Multi-stage build: Composer install → PHP-FPM runtime
# ==============================================================================

# Stage 1: Install dependencies
FROM composer:2 AS dependencies

WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-interaction --prefer-dist --optimize-autoloader --no-dev --no-scripts

# Stage 2: Runtime
FROM php:8.3-fpm-alpine

# Install PHP extensions
RUN apk add --no-cache \
        libzip-dev \
        libpng-dev \
        libjpeg-turbo-dev \
        freetype-dev \
        oniguruma-dev \
        icu-dev \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install \
        pdo \
        pdo_mysql \
        zip \
        gd \
        mbstring \
        intl \
        bcmath \
        opcache

# Install Redis extension
RUN apk add --no-cache --virtual .build-deps $PHPIZE_DEPS \
    && pecl install redis \
    && docker-php-ext-enable redis \
    && apk del .build-deps

# PHP configuration
COPY docker/php.ini /usr/local/etc/php/conf.d/app.ini

WORKDIR /var/www/html

# Copy application code
COPY . .
COPY --from=dependencies /app/vendor vendor/

# Set permissions
RUN chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

# Generate optimized autoload
RUN composer dump-autoload --optimize --no-dev 2>/dev/null || true

EXPOSE 9000

CMD ["php-fpm"]
