# frozen_string_literal: trueo

require 'gitlab-dangerfiles'

Gitlab::Dangerfiles.for_project(self, 'gitlab-cli') do |dangerfiles|
  dangerfiles.import_plugins
  dangerfiles.import_dangerfiles(only: %w[simple_roulette])
end
