from django.core.management.base import BaseCommand
from imagesManager.tasks import check_update


class Command(BaseCommand):
    help = 'Runs the check_update task'

    def handle(self, *args, **options):
        check_update()
